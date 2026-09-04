package tui

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathfavour/beaverish/config"
	"github.com/nathfavour/beaverish/pkg/engine"
	"github.com/nathfavour/beaverish/pkg/types"
)

var (
	// Palettes & Colors
	subtle    = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#555555"}
	highlight = lipgloss.AdaptiveColor{Light: "#7D56F4", Dark: "#874BFD"}
	special   = lipgloss.AdaptiveColor{Light: "#2ECC71", Dark: "#43BF6D"}
	warnColor = lipgloss.AdaptiveColor{Light: "#FFA500", Dark: "#FFB86C"}
	danger    = lipgloss.AdaptiveColor{Light: "#E74C3C", Dark: "#FF5555"}
	cyanColor = lipgloss.AdaptiveColor{Light: "#00BCD4", Dark: "#00E5FF"}
	white     = lipgloss.Color("#FFFFFF")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(highlight).
			Padding(0, 2).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cyanColor).
			Padding(0, 1)

	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(highlight).
			Padding(0, 1)

	logBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(0, 1)

	dimStyle = lipgloss.NewStyle().Foreground(subtle)
)

type tickMsg time.Time

type tradeLog struct {
	time    string
	message string
}

type Model struct {
	cfg        *config.Config
	eng        *engine.Engine
	markets    []types.MarketSnapshot
	account    *types.AccountStatus
	positions  []types.Position
	logs       []tradeLog
	lastSweep  time.Time
	statusMsg  string
	quitting   bool
	width      int
	height     int
	cursor     int
}

func NewModel(cfg *config.Config, eng *engine.Engine) Model {
	return Model{
		cfg:     cfg,
		eng:     eng,
		markets: []types.MarketSnapshot{},
		account: &types.AccountStatus{
			Address:           cfg.Wallet.Address,
			NativeBalance:     big.NewInt(0),
			CollateralBalance: big.NewInt(0),
		},
		positions: []types.Position{},
		logs: []tradeLog{
			{time: time.Now().Format("15:04:05"), message: "Engine connected to Somnia Shannon testnet"},
			{time: time.Now().Format("15:04:05"), message: "Automated parity arbitrage & momentum evaluation active"},
		},
		lastSweep: time.Now(),
		statusMsg: "Ready — Press [u] Buy UP, [d] Buy DOWN, [s] Sweep",
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) addLog(msg string) {
	entry := tradeLog{
		time:    time.Now().Format("15:04:05"),
		message: msg,
	}
	m.logs = append(m.logs, entry)
	if len(m.logs) > 8 {
		m.logs = m.logs[len(m.logs)-8:]
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.markets)-1 {
				m.cursor++
			}
		case "u", "U":
			// Execute manual UP trade on currently selected market
			if len(m.markets) > 0 && m.cursor < len(m.markets) {
				market := m.markets[m.cursor]
				oneUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
				betSizeFloat := new(big.Float).Mul(big.NewFloat(m.cfg.Risk.MaxBetSizeUnits), new(big.Float).SetInt(oneUnit))
				betSizeWei, _ := betSizeFloat.Int(nil)

				sig := types.Signal{
					ShouldExecute: true,
					MarketID:      market.MarketID,
					TargetSide:    types.OutcomeUp,
					TargetPrice:   market.BestAskUp,
					PositionSize:  betSizeWei,
					Reason:        "Interactive Manual Execution: BUY UP",
				}
				m.eng.Executor().Dispatch(sig)
				m.statusMsg = fmt.Sprintf("dispatched BUY UP on %s", market.UnderlyingAsset)
				m.addLog(fmt.Sprintf("Manual Order: BUY UP on %s at %s", market.UnderlyingAsset, formatPrice(market.BestAskUp)))
			}
		case "d", "D":
			// Execute manual DOWN trade on currently selected market
			if len(m.markets) > 0 && m.cursor < len(m.markets) {
				market := m.markets[m.cursor]
				oneUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
				betSizeFloat := new(big.Float).Mul(big.NewFloat(m.cfg.Risk.MaxBetSizeUnits), new(big.Float).SetInt(oneUnit))
				betSizeWei, _ := betSizeFloat.Int(nil)

				sig := types.Signal{
					ShouldExecute: true,
					MarketID:      market.MarketID,
					TargetSide:    types.OutcomeDown,
					TargetPrice:   market.BestAskDown,
					PositionSize:  betSizeWei,
					Reason:        "Interactive Manual Execution: BUY DOWN",
				}
				m.eng.Executor().Dispatch(sig)
				m.statusMsg = fmt.Sprintf("dispatched BUY DOWN on %s", market.UnderlyingAsset)
				m.addLog(fmt.Sprintf("Manual Order: BUY DOWN on %s at %s", market.UnderlyingAsset, formatPrice(market.BestAskDown)))
			}
		case "s", "S":
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			txs, err := m.eng.Settler().Sweep(ctx)
			if err != nil {
				m.statusMsg = fmt.Sprintf("Sweep error: %v", err)
				m.addLog(fmt.Sprintf("Sweep error: %v", err))
			} else {
				m.statusMsg = fmt.Sprintf("Manual Sweep: %d claims executed", len(txs))
				m.addLog(fmt.Sprintf("Manual Sweep: %d claims redeemed", len(txs)))
			}
			m.lastSweep = time.Now()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if markets, err := m.eng.Adapter().GetActiveMarkets(ctx); err == nil {
			m.markets = markets
		}
		if acc, err := m.eng.Adapter().GetAccountStatus(ctx); err == nil {
			m.account = acc
		}
		if positions, err := m.eng.Adapter().GetPositions(ctx); err == nil {
			m.positions = positions
		}
		return m, tickCmd()
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return "Shutting down Beaverish...\n"
	}

	var b strings.Builder

	// Top Title Banner
	b.WriteString(titleStyle.Render("🦫 Beaverish Terminal — High-Throughput DreamDEX Arbitrage Engine"))
	b.WriteString("\n\n")

	// Account & Network Status Card
	oneUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	oneUnitFloat := new(big.Float).SetInt(oneUnit)

	nativeBal := big.NewInt(0)
	colBal := big.NewInt(0)
	addrStr := m.cfg.Wallet.Address
	if m.account != nil {
		if m.account.NativeBalance != nil {
			nativeBal = m.account.NativeBalance
		}
		if m.account.CollateralBalance != nil {
			colBal = m.account.CollateralBalance
		}
		if m.account.Address != "" {
			addrStr = m.account.Address
		}
	}

	nativeFloat := new(big.Float).Quo(new(big.Float).SetInt(nativeBal), oneUnitFloat)
	colFloat := new(big.Float).Quo(new(big.Float).SetInt(colBal), oneUnitFloat)

	accInfo := fmt.Sprintf(
		"Address: %s  |  Chain ID: %d (%s)\nNative Gas: %.4f STT  |  Collateral: %.2f Units  |  Open Positions: %d",
		addrStr,
		m.cfg.Network.ChainID,
		m.cfg.Network.Driver,
		nativeFloat,
		colFloat,
		len(m.positions),
	)
	b.WriteString(borderStyle.Render(accInfo))
	b.WriteString("\n\n")

	// Market Watcher Section
	b.WriteString(headerStyle.Render("Active Event Contracts (Use [↑/↓] to navigate, [u] Buy UP, [d] Buy DOWN):"))
	b.WriteString("\n")

	if len(m.markets) == 0 {
		b.WriteString(dimStyle.Render("  No active event markets discovered.\n"))
	} else {
		for i, market := range m.markets {
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}

			askUpStr := formatPrice(market.BestAskUp)
			askDownStr := formatPrice(market.BestAskDown)

			timeRemaining := market.ExpiryTimestamp - time.Now().Unix()
			timeStr := fmt.Sprintf("%ds", timeRemaining)
			if timeRemaining < 0 {
				timeStr = "EXPIRED"
			}

			// Parity Edge Indicator
			edgeStr := ""
			if market.BestAskUp != nil && market.BestAskDown != nil {
				sumAsks := new(big.Int).Add(market.BestAskUp, market.BestAskDown)
				if sumAsks.Cmp(oneUnit) < 0 {
					edgeVal := new(big.Int).Sub(oneUnit, sumAsks)
					edgeF, _ := new(big.Float).Quo(new(big.Float).SetInt(edgeVal), oneUnitFloat).Float64()
					edgeStr = fmt.Sprintf(" | Arb Edge: +%.1f%%", edgeF*100)
				}
			}

			line := fmt.Sprintf(
				"%s[%s] Asset: %-4s | Strike: $%s | UP Ask: %s | DOWN Ask: %s | Expiry: %-7s | Status: %s%s",
				prefix,
				market.MarketID[:10]+"...",
				market.UnderlyingAsset,
				formatStrike(market.StrikePrice),
				askUpStr,
				askDownStr,
				timeStr,
				market.Status.String(),
				edgeStr,
			)

			if i == m.cursor {
				b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(highlight).Render(line) + "\n")
			} else {
				b.WriteString(line + "\n")
			}
		}
	}

	// Recent Activity Log Box
	b.WriteString("\n")
	b.WriteString(headerStyle.Render("Live Activity Feed:"))
	b.WriteString("\n")
	var logContent strings.Builder
	if len(m.logs) == 0 {
		logContent.WriteString(dimStyle.Render("  No activity recorded yet.\n"))
	} else {
		for _, l := range m.logs {
			logContent.WriteString(fmt.Sprintf("  [%s] %s\n", l.time, l.message))
		}
	}
	b.WriteString(logBoxStyle.Render(logContent.String()))
	b.WriteString("\n")

	// Status & Shortcuts footer
	b.WriteString(dimStyle.Render(fmt.Sprintf("Status: %s  |  [u] Buy UP  [d] Buy DOWN  [s] Sweep  [q] Quit", m.statusMsg)))
	b.WriteString("\n")

	return b.String()
}

func formatPrice(price *big.Int) string {
	if price == nil {
		return "N/A"
	}
	oneUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	f, _ := new(big.Float).Quo(new(big.Float).SetInt(price), new(big.Float).SetInt(oneUnit)).Float64()
	return fmt.Sprintf("%.3f", f)
}

func formatStrike(strike *big.Int) string {
	if strike == nil {
		return "0.00"
	}
	f, _ := new(big.Float).Quo(new(big.Float).SetInt(strike), big.NewFloat(1e6)).Float64()
	return fmt.Sprintf("%.2f", f)
}

func RunTUI(cfg *config.Config, eng *engine.Engine) error {
	p := tea.NewProgram(NewModel(cfg, eng))
	_, err := p.Run()
	return err
}
