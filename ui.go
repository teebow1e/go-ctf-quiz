package main

import (
	"fmt"
	"net"
	"time"
)

// ANSI color codes
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"

	ColorBoldRed     = "\033[1;31m"
	ColorBoldGreen   = "\033[1;32m"
	ColorBoldYellow  = "\033[1;33m"
	ColorBoldBlue    = "\033[1;34m"
	ColorBoldMagenta = "\033[1;35m"
	ColorBoldCyan    = "\033[1;36m"
	ColorBoldWhite   = "\033[1;37m"

	// Background colors
	BgRed    = "\033[41m"
	BgGreen  = "\033[42m"
	BgYellow = "\033[43m"
	BgBlue   = "\033[44m"
)

// UI formatting helpers
func colorize(color, text string) string {
	return color + text + ColorReset
}

func banner(text string) string {
	line := "═══════════════════════════════════════════════════════════════════════════════"
	return fmt.Sprintf("\n%s\n%s\n%s\n",
		colorize(ColorBoldCyan, line),
		colorize(ColorBoldCyan, center(text, 79)),
		colorize(ColorBoldCyan, line))
}

func center(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := (width - len(text)) / 2
	return fmt.Sprintf("%*s%s", padding, "", text)
}

func success(text string) string {
	return colorize(ColorBoldGreen, "✓ "+text)
}

func failure(text string) string {
	return colorize(ColorBoldRed, "✗ "+text)
}

func warning(text string) string {
	return colorize(ColorBoldYellow, "⚠ "+text)
}

func info(text string) string {
	return colorize(ColorBoldCyan, "ℹ "+text)
}

func prompt(text string) string {
	return colorize(ColorBoldBlue, "► "+text)
}

func highlightBox(text string) string {
	topBottom := "╔═══════════════════════════════════════════════════════════════════════════════╗"
	middle := fmt.Sprintf("║ %-77s ║", text)
	return colorize(ColorBoldMagenta, topBottom+"\n"+middle+"\n"+topBottom)
}

func divider() string {
	return colorize(ColorCyan, "───────────────────────────────────────────────────────────────────────────────")
}

func showFlag(conn net.Conn, flag string) {
	conn.Write([]byte("\n\n"))
	conn.Write([]byte(colorize(ColorBoldGreen, "═══════════════════════════════════════════════════════════════════════════════") + "\n"))
	conn.Write([]byte(colorize(ColorBoldGreen, "                           🎉 CONGRATULATIONS! 🎉                              ") + "\n"))
	conn.Write([]byte(colorize(ColorBoldGreen, "═══════════════════════════════════════════════════════════════════════════════") + "\n\n"))
	conn.Write([]byte(colorize(ColorBoldWhite, "   You've successfully conquered all the challenges!\n")))
	time.Sleep(1 * time.Second)
	conn.Write([]byte(colorize(ColorBoldCyan, "   🚩 Here is your flag:\n\n")))
	conn.Write([]byte(colorize(ColorBoldYellow, "      "+flag+"\n\n")))
	conn.Write([]byte(divider() + "\n\n"))
}
