package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/pavelc4/astra/cmd/cookie-dumper/extractor"
)

const usage = `cookie-dumper — extract Instagram / Facebook / Pixiv cookies from your browser

Usage:
  cookie-dumper [options]

Options:
  -browser   firefox|chromium          Browser backend (default: firefox)
  -platform  instagram|facebook|pixiv  Platform (default: instagram)
  -profile   /path/to/profile          Override profile location (optional)
  -exec      /path/to/binary           Override browser binary (optional, chromium only)
  -out       export|env|raw            Output format (default: export)
  -write-env                           Write/update directly to .env file

Example:
  cookie-dumper
  cookie-dumper -write-env
  cookie-dumper -platform pixiv -write-env
  cookie-dumper -platform facebook -write-env
  cookie-dumper -browser chromium -exec /usr/bin/brave-browser -out raw
`

type platformMeta struct {
	label      string
	envVar     string
	cookieKeys []string
}

var platformConfigs = map[string]platformMeta{
	"instagram": {
		label:      "Instagram",
		envVar:     "INSTAGRAM_COOKIES",
		cookieKeys: []string{"sessionid", "csrftoken"},
	},
	"facebook": {
		label:      "Facebook",
		envVar:     "FACEBOOK_COOKIES",
		cookieKeys: []string{"c_user", "xs", "datr", "sb", "fr", "dpr"},
	},
	"pixiv": {
		label:      "Pixiv",
		envVar:     "PIXIV_COOKIES",
		cookieKeys: []string{"PHPSESSID", "login_ever", "device_token", "p_ab_id", "cf_clearance"},
	},
}

func main() {
	var (
		browserFlag  = flag.String("browser", "firefox", "Browser backend: firefox | chromium")
		platformFlag = flag.String("platform", "instagram", "Platform: instagram | facebook | pixiv")
		profileFlag  = flag.String("profile", "", "Path to browser profile")
		execFlag     = flag.String("exec", "", "Path to browser binary")
		outFlag      = flag.String("out", "export", "Output format: export | env | raw")
		writeEnvFlag = flag.Bool("write-env", false, "Write extracted cookies directly to .env file")
	)

	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	platform := strings.ToLower(strings.TrimSpace(*platformFlag))
	meta, ok := platformConfigs[platform]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: unknown platform '%s'. Use 'instagram', 'facebook' or 'pixiv'\n", platform)
		os.Exit(1)
	}

	fmt.Printf("=== %s Cookie Extractor ===\n\n", meta.label)

	ex, err := createExtractor(*browserFlag, *profileFlag, *execFlag, platform)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cookies, err := ex.Extract()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}

	cookieStr := formatCookieString(cookies, meta)

	fmt.Printf("\nCookies extracted successfully!\n\n")

	outputResult(*outFlag, meta.envVar, cookieStr)

	if *writeEnvFlag {
		if err := updateEnvFile(".env", meta.envVar, cookieStr); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to .env: %v\n", err)
		} else {
			fmt.Printf("\n[.env] Successfully updated %s in .env\n", meta.envVar)
		}
	}
}

func createExtractor(browser, profile, exec, platform string) (extractor.Extractor, error) {
	switch strings.ToLower(browser) {
	case "firefox":
		return extractor.NewFirefox(profile, platform), nil
	case "chromium":
		return extractor.NewChromiumExtractor(profile, exec, platform)
	default:
		return nil, fmt.Errorf("unknown browser '%s'. Use 'firefox' or 'chromium'", browser)
	}
}

func outputResult(outFormat, envVar, cookieStr string) {
	switch outFormat {
	case "export":
		fmt.Printf("export %s=\"%s\"\n", envVar, cookieStr)
	case "env":
		fmt.Printf("%s=\"%s\"\n", envVar, cookieStr)
	case "raw":
		fmt.Println(cookieStr)
	default:
		fmt.Fprintf(os.Stderr, "Warning: unknown format '%s', falling back to 'export'\n\n", outFormat)
		fmt.Printf("export %s=\"%s\"\n", envVar, cookieStr)
	}
}

func formatCookieString(c *extractor.Cookies, meta platformMeta) string {
	var parts []string
	for _, name := range meta.cookieKeys {
		if v, ok := c.Values[name]; ok && v != "" {
			parts = append(parts, name+"="+v)
		}
	}
	return strings.Join(parts, "; ")
}

func updateEnvFile(envPath, envVar, cookieStr string) error {
	content, err := os.ReadFile(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(envPath, []byte(fmt.Sprintf("%s=%q\n", envVar, cookieStr)), 0644)
		}
		return err
	}

	lines := strings.Split(string(content), "\n")
	found := false
	prefix := envVar + "="

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			lines[i] = fmt.Sprintf("%s=%q", envVar, cookieStr)
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, fmt.Sprintf("%s=%q", envVar, cookieStr))
	}

	return os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0644)
}
