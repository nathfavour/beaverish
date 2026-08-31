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
	// Styles
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	warnColor = lipgloss.AdaptiveColor{Light: "#FFA500", Dark: "#FFB86C"}

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(highlight).
			Padding(0, 1).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(special).
			Padding(0, 1)

	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(highlight).
			Padding(1, 2)

	dimStyle = lipgloss.NewStyle().Foreground(subtle)
)

type tickMsg time.Time

type Model struct {
	cfg        *config.Config
	eng        *engine.Engine
	markets    []types.MarketSnapshot
	account    *types.AccountStatus
	lastSweep  time.Time
	statusMsg  string
	quitting   bool
	width      int
	height     int
	cursor     int
}

func NewModel(cfg *config.Config, eng *engine.Engine) Model {
	return Model{
		cfg:       cfg,
		eng:       eng,
		markets:   []types.MarketSnapshot{},
		account:   &types.AccountStatus{},
		lastSweep: time.Now(),
		statusMsg: "Live Engine Connected",
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
		case "s":
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			txs, err := m.eng.Settler().Sweep(ctx)
			if err != nil {
				m.statusMsg = fmt.Sprintf("Sweep error: %v", err)
			} else {
				m.statusMsg = fmt.Sprintf("Manual Sweep: %d claims executed", len(txs))
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
	b.WriteString(titleStyle.Render("🦫 Beaverish Terminal — DreamDEX / Somnia EVM Market Engine"))
	b.WriteString("\n\n")

	// Account & Network Status Card
	oneUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	oneUnitFloat := new(big.Float).SetInt(oneUnit)

	nativeFloat := new(big.Float).Quo(new(big.Float).SetInt(m.account.NativeBalance), oneUnitFloat)
	colFloat := new(big.Float).Quo(new(big.Float).SetInt(m.account.CollateralBalance), oneUnitFloat)

	accInfo := fmt.Sprintf(
		"Address: %s  |  Chain ID: %d (%s)\nNative Gas: %.4f STT  |  Collateral: %.2f Units  |  Open Positions: %d",
		m.account.Address,
		m.cfg.Network.ChainID,
		m.cfg.Network.Driver,
		nativeFloat,
		colFloat,
		m.account.OpenPositions,
	)
	b.WriteString(borderStyle.Render(accInfo))
	b.WriteString("\n\n")

	// Market Watcher Section
	b.WriteString(headerStyle.Render("Active Event Contracts:"))
	b.WriteString("\n")

	if len(m.markets) == 0 {
		b.WriteString(dimStyle.Render("  No active event markets discovered.\n"))
	} else {
		for i, market := range m.markets {
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}

			askUpStr := "N/A"
			if market.BestAskUp != nil {
				f, _ := new(big.Float).Quo(new(big.Float).SetInt(market.BestAskUp), oneUnitFloat).Float64()
				askUpStr = fmt.Sprintf("%.3f", f)
			}

			askDownStr := "N/A"
			if market.BestAskDown != nil {
				f, _ := new(big.Float).Quo(new(big.Float).SetInt(market.BestAskDown), oneUnitFloat).Float64()
				askDownStr = fmt.Sprintf("%.3f", f)
			}

			timeRemaining := market.ExpiryTimestamp - time.Now().Unix()
			timeStr := fmt.Sprintf("%ds", timeRemaining)
			if timeRemaining < 0 {
				timeStr = "EXPIRED"
			}

			line := fmt.Sprintf(
				"%s[%s] Asset: %-4s | Strike: $%s | UP Ask: %s | DOWN Ask: %s | Expiry: %-7s | Status: %s",
				prefix,
				market.MarketID[:10]+"...",
				market.UnderlyingAsset,
				formatStrike(market.StrikePrice),
				askUpStr,
				askDownStr,
				timeStr,
				market.Status.String(),
			)

			if i == m.cursor {
				b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(highlight).Render(line) + "\n")
			} else {
				b.WriteString(line + "\n")
			}
		}
	}

	// Status & Shortcuts footer
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("Status: %s  |  [s] Manual Sweep  [q] Quit", m.statusMsg)))
	b.WriteString("\n")

	return b.String()
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
