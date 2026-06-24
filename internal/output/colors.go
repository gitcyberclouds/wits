package output

// Bright ANSI codes (90-97) replace standard (30-37) for contrast on dark themes.
var (
	ColorReset     = ansiString("\033[0m")
	ColorRed       = ansiString("\033[91m")
	ColorGreen     = ansiString("\033[92m")
	ColorBlue      = ansiString("\033[94m")
	ColorCyan      = ansiString("\033[96m")
	ColorMagenta   = ansiString("\033[95m")
	ColorDim       = ansiString("\033[90m")
	ColorDimYellow = ansiString("\033[93m")
)
