package ui

import "github.com/charmbracelet/lipgloss"

const (
	ThemeModeDark  = "dark"
	ThemeModeLight = "light"
)

// Theme holds the active colour palette.
type Theme struct {
	Name      string
	Mode      string // "dark" or "light"
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Success   lipgloss.Color
	Warn      lipgloss.Color
	Error     lipgloss.Color
	Muted     lipgloss.Color
	Text      lipgloss.Color
	Bg        lipgloss.Color
	Surface   lipgloss.Color
}

// CurrentTheme is the active theme. It can be changed at runtime.
var CurrentTheme = TokyoNightTheme()

// All themes in display order.
var themes = []Theme{
	TokyoNightTheme(),
	TokyoNightStormTheme(),
	NordTheme(),
	CatppuccinMochaTheme(),
	RosePineTheme(),
	GruvboxDarkTheme(),
	EverforestDarkTheme(),
	SolarizedDarkTheme(),
	LightTheme(),
	CatppuccinLatteTheme(),
	RosePineDawnTheme(),
	SolarizedLightTheme(),
	EverforestLightTheme(),
}

// TokyoNightTheme is the default dark theme.
func TokyoNightTheme() Theme {
	return Theme{
		Name:      "tokyo-night",
		Mode:      ThemeModeDark,
		Primary:   lipgloss.Color("#7AA2F7"),
		Secondary: lipgloss.Color("#BB9AF7"),
		Success:   lipgloss.Color("#9ECE6A"),
		Warn:      lipgloss.Color("#E0AF68"),
		Error:     lipgloss.Color("#F7768E"),
		Muted:     lipgloss.Color("#565F89"),
		Text:      lipgloss.Color("#C0CAF5"),
		Bg:        lipgloss.Color("#1A1B26"),
		Surface:   lipgloss.Color("#24283B"),
	}
}

// TokyoNightStormTheme is a deeper, stormier variant.
func TokyoNightStormTheme() Theme {
	return Theme{
		Name:      "tokyo-night-storm",
		Mode:      ThemeModeDark,
		Primary:   lipgloss.Color("#7AA2F7"),
		Secondary: lipgloss.Color("#BB9AF7"),
		Success:   lipgloss.Color("#9ECE6A"),
		Warn:      lipgloss.Color("#E0AF68"),
		Error:     lipgloss.Color("#F7768E"),
		Muted:     lipgloss.Color("#565F89"),
		Text:      lipgloss.Color("#C0CAF5"),
		Bg:        lipgloss.Color("#11121D"),
		Surface:   lipgloss.Color("#1B1E2E"),
	}
}

// NordTheme is a cool arctic palette.
func NordTheme() Theme {
	return Theme{
		Name:      "nord",
		Mode:      ThemeModeDark,
		Primary:   lipgloss.Color("#88C0D0"),
		Secondary: lipgloss.Color("#B48EAD"),
		Success:   lipgloss.Color("#A3BE8C"),
		Warn:      lipgloss.Color("#EBCB8B"),
		Error:     lipgloss.Color("#BF616A"),
		Muted:     lipgloss.Color("#616E88"),
		Text:      lipgloss.Color("#D8DEE9"),
		Bg:        lipgloss.Color("#2E3440"),
		Surface:   lipgloss.Color("#3B4252"),
	}
}

// CatppuccinMochaTheme is a soft pastel dark theme.
func CatppuccinMochaTheme() Theme {
	return Theme{
		Name:      "catppuccin-mocha",
		Mode:      ThemeModeDark,
		Primary:   lipgloss.Color("#89B4FA"),
		Secondary: lipgloss.Color("#CBA6F7"),
		Success:   lipgloss.Color("#A6E3A1"),
		Warn:      lipgloss.Color("#F9E2AF"),
		Error:     lipgloss.Color("#F38BA8"),
		Muted:     lipgloss.Color("#6C7086"),
		Text:      lipgloss.Color("#CDD6F4"),
		Bg:        lipgloss.Color("#1E1E2E"),
		Surface:   lipgloss.Color("#313244"),
	}
}

// RosePineTheme is a muted earthy dark theme.
func RosePineTheme() Theme {
	return Theme{
		Name:      "rose-pine",
		Mode:      ThemeModeDark,
		Primary:   lipgloss.Color("#9CCFD8"),
		Secondary: lipgloss.Color("#C4A7E7"),
		Success:   lipgloss.Color("#ABE9B3"),
		Warn:      lipgloss.Color("#F6C177"),
		Error:     lipgloss.Color("#EB6F92"),
		Muted:     lipgloss.Color("#6E6A86"),
		Text:      lipgloss.Color("#E0DEF4"),
		Bg:        lipgloss.Color("#191724"),
		Surface:   lipgloss.Color("#26233A"),
	}
}

// GruvboxDarkTheme is a warm retro dark theme.
func GruvboxDarkTheme() Theme {
	return Theme{
		Name:      "gruvbox-dark",
		Mode:      ThemeModeDark,
		Primary:   lipgloss.Color("#83A598"),
		Secondary: lipgloss.Color("#D3869B"),
		Success:   lipgloss.Color("#B8BB26"),
		Warn:      lipgloss.Color("#FABD2F"),
		Error:     lipgloss.Color("#FB4934"),
		Muted:     lipgloss.Color("#928374"),
		Text:      lipgloss.Color("#EBDBB2"),
		Bg:        lipgloss.Color("#282828"),
		Surface:   lipgloss.Color("#3C3836"),
	}
}

// EverforestDarkTheme is a muted nature dark theme.
func EverforestDarkTheme() Theme {
	return Theme{
		Name:      "everforest-dark",
		Mode:      ThemeModeDark,
		Primary:   lipgloss.Color("#A7C080"),
		Secondary: lipgloss.Color("#D699B6"),
		Success:   lipgloss.Color("#83C092"),
		Warn:      lipgloss.Color("#DBBC7F"),
		Error:     lipgloss.Color("#E67E80"),
		Muted:     lipgloss.Color("#7A8478"),
		Text:      lipgloss.Color("#D3C6AA"),
		Bg:        lipgloss.Color("#232A2E"),
		Surface:   lipgloss.Color("#2D353B"),
	}
}

// SolarizedDarkTheme is the classic low-saturation dark theme.
func SolarizedDarkTheme() Theme {
	return Theme{
		Name:      "solarized-dark",
		Mode:      ThemeModeDark,
		Primary:   lipgloss.Color("#268BD2"),
		Secondary: lipgloss.Color("#6C71C4"),
		Success:   lipgloss.Color("#859900"),
		Warn:      lipgloss.Color("#B58900"),
		Error:     lipgloss.Color("#DC322F"),
		Muted:     lipgloss.Color("#586E75"),
		Text:      lipgloss.Color("#839496"),
		Bg:        lipgloss.Color("#002B36"),
		Surface:   lipgloss.Color("#073642"),
	}
}

// LightTheme is a smooth light alternative.
func LightTheme() Theme {
	return Theme{
		Name:      "light",
		Mode:      ThemeModeLight,
		Primary:   lipgloss.Color("#5B7C99"),
		Secondary: lipgloss.Color("#8E7DBE"),
		Success:   lipgloss.Color("#6A9C6E"),
		Warn:      lipgloss.Color("#C59B6D"),
		Error:     lipgloss.Color("#B95C5C"),
		Muted:     lipgloss.Color("#9CA3AF"),
		Text:      lipgloss.Color("#374151"),
		Bg:        lipgloss.Color("#F5F5F0"),
		Surface:   lipgloss.Color("#E8E8E2"),
	}
}

// CatppuccinLatteTheme is a soft pastel light theme.
func CatppuccinLatteTheme() Theme {
	return Theme{
		Name:      "catppuccin-latte",
		Mode:      ThemeModeLight,
		Primary:   lipgloss.Color("#1E66F5"),
		Secondary: lipgloss.Color("#8839EF"),
		Success:   lipgloss.Color("#40A02B"),
		Warn:      lipgloss.Color("#DF8E1D"),
		Error:     lipgloss.Color("#D20F39"),
		Muted:     lipgloss.Color("#9CA0B0"),
		Text:      lipgloss.Color("#4C4F69"),
		Bg:        lipgloss.Color("#EFF1F5"),
		Surface:   lipgloss.Color("#CCD0DA"),
	}
}

// RosePineDawnTheme is a warm morning light theme.
func RosePineDawnTheme() Theme {
	return Theme{
		Name:      "rose-pine-dawn",
		Mode:      ThemeModeLight,
		Primary:   lipgloss.Color("#286983"),
		Secondary: lipgloss.Color("#907AA9"),
		Success:   lipgloss.Color("#618774"),
		Warn:      lipgloss.Color("#EA9D34"),
		Error:     lipgloss.Color("#B4637A"),
		Muted:     lipgloss.Color("#9893A5"),
		Text:      lipgloss.Color("#575279"),
		Bg:        lipgloss.Color("#FAF4ED"),
		Surface:   lipgloss.Color("#F2E9E1"),
	}
}

// SolarizedLightTheme is the classic low-saturation light theme.
func SolarizedLightTheme() Theme {
	return Theme{
		Name:      "solarized-light",
		Mode:      ThemeModeLight,
		Primary:   lipgloss.Color("#268BD2"),
		Secondary: lipgloss.Color("#6C71C4"),
		Success:   lipgloss.Color("#859900"),
		Warn:      lipgloss.Color("#B58900"),
		Error:     lipgloss.Color("#DC322F"),
		Muted:     lipgloss.Color("#93A1A1"),
		Text:      lipgloss.Color("#586E75"),
		Bg:        lipgloss.Color("#FDF6E3"),
		Surface:   lipgloss.Color("#EEE8D5"),
	}
}

// EverforestLightTheme is a muted nature light theme.
func EverforestLightTheme() Theme {
	return Theme{
		Name:      "everforest-light",
		Mode:      ThemeModeLight,
		Primary:   lipgloss.Color("#5F8B6A"),
		Secondary: lipgloss.Color("#B279A7"),
		Success:   lipgloss.Color("#4F8B6C"),
		Warn:      lipgloss.Color("#B5894E"),
		Error:     lipgloss.Color("#C07775"),
		Muted:     lipgloss.Color("#A4AD9E"),
		Text:      lipgloss.Color("#5C6F5B"),
		Bg:        lipgloss.Color("#F3EAD3"),
		Surface:   lipgloss.Color("#E4DCC9"),
	}
}

// ThemeNames returns the available preset theme names.
func ThemeNames() []string {
	names := make([]string, 0, len(themes))
	for _, t := range themes {
		names = append(names, t.Name)
	}
	return names
}

// ThemeNamesByMode returns theme names filtered by light/dark mode.
func ThemeNamesByMode(mode string) []string {
	names := make([]string, 0)
	for _, t := range themes {
		if t.Mode == mode {
			names = append(names, t.Name)
		}
	}
	return names
}

// ThemeByName returns a preset theme by name, falling back to the default.
func ThemeByName(name string) Theme {
	for _, t := range themes {
		if t.Name == name {
			return t
		}
	}
	return TokyoNightTheme()
}

// ThemesByMode returns all themes matching the given mode.
func ThemesByMode(mode string) []Theme {
	out := make([]Theme, 0)
	for _, t := range themes {
		if t.Mode == mode {
			out = append(out, t)
		}
	}
	return out
}

// DefaultThemeForMode returns the first theme of the requested mode.
func DefaultThemeForMode(mode string) Theme {
	for _, t := range themes {
		if t.Mode == mode {
			return t
		}
	}
	return TokyoNightTheme()
}

// SetTheme updates the global colour variables and recreates all styles.
func SetTheme(t Theme) {
	CurrentTheme = t
	recreateStyles()
}

// Colour accessors for use outside of the pre-built styles.
func ColorPrimary() lipgloss.Color   { return CurrentTheme.Primary }
func ColorSecondary() lipgloss.Color { return CurrentTheme.Secondary }
func ColorSuccess() lipgloss.Color   { return CurrentTheme.Success }
func ColorWarn() lipgloss.Color      { return CurrentTheme.Warn }
func ColorError() lipgloss.Color     { return CurrentTheme.Error }
func ColorMuted() lipgloss.Color     { return CurrentTheme.Muted }
func ColorText() lipgloss.Color      { return CurrentTheme.Text }
func ColorBg() lipgloss.Color        { return CurrentTheme.Bg }
func ColorSurface() lipgloss.Color   { return CurrentTheme.Surface }
