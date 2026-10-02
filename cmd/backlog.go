package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/geomark27/deploy-doc/internal/backlog"
	"github.com/geomark27/deploy-doc/internal/config"
)

var backlogSubcommands = map[string]func([]string) error{
	"scan": runBacklogScan,
}

var backlogScanShortFlags = map[string]string{
	"-p": "--project",
	"-r": "--repo",
	"-l": "--limit",
	"-o": "--output",
}

const defaultBacklogLimit = 10

func runBacklog(args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printBacklogUsage()
		return nil
	}
	sub := args[0]
	fn, ok := backlogSubcommands[sub]
	if !ok {
		fmt.Fprintf(os.Stderr, clr(clRed, "Subcomando desconocido: ")+"backlog %s\n\n", sub)
		printBacklogUsage()
		return fmt.Errorf("subcomando inválido")
	}
	return fn(args[1:])
}

// runBacklogScan detects technical debt in the project's repos. It only reads
// them, so it uses config.LoadLocal and works without Atlassian credentials.
func runBacklogScan(args []string) error {
	flags := parseFlagsWithShorts(args, backlogScanShortFlags)
	_, asJSON := flags["--json"]
	_, withDeps := flags["--deps"]
	outputFlag, saveReport := flags["--output"]

	cfg, err := config.LoadLocal()
	if err != nil {
		return err
	}
	proj, projName, err := cfg.GetProject(flags["--project"])
	if err != nil {
		return err
	}
	targets, err := resolveBacklogTargets(flags["--path"], flags["--repo"], proj, projName)
	if err != nil {
		return err
	}
	limit, err := parseBacklogLimit(flags["--limit"])
	if err != nil {
		return err
	}
	if saveReport {
		if err := validateBacklogOutput(outputFlag, len(targets)); err != nil {
			return err
		}
	}

	var reports []*backlog.Report
	var failed []string
	for i, t := range targets {
		if !asJSON {
			stepLabel(i+1, len(targets), "Analizando "+t.describe(projName, flags["--path"] != "")+"...")
		}

		opts := backlogOptions(proj.BacklogFor(t.repo), flags["--since"])
		rep, err := backlog.Scan(t.dir, backlog.ScanOptions{Options: opts, Dependencies: withDeps})
		if err != nil {
			if len(targets) == 1 {
				return err
			}
			reportBacklogFailure(asJSON, t.repo, err)
			failed = append(failed, t.repo)
			continue
		}

		// The saved file keeps every finding; --limit only trims what is shown.
		savedTo := ""
		if saveReport {
			name := backlogReportName(projName, t.dir, t.repo)
			if savedTo, err = saveBacklogReport(rep, backlogOutputPath(outputFlag, len(targets), name), name); err != nil {
				return err
			}
		}
		if limit > 0 && len(rep.Findings) > limit {
			rep.Findings = rep.Findings[:limit]
		}
		reports = append(reports, rep)

		if !asJSON {
			printBacklogReport(rep)
			if savedTo != "" {
				okLine("Reporte completo guardado en " + savedTo)
			} else {
				fmt.Println("Usa " + clBold + "-o" + clReset + " para guardar el reporte completo en JSON.")
			}
			fmt.Println()
		}
	}

	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(reports); err != nil {
			return err
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("no se pudo analizar: %s", strings.Join(failed, ", "))
	}
	return nil
}

// backlogTarget is one repo to scan.
type backlogTarget struct {
	repo string // "backend" or "frontend"
	dir  string
}

// describe names the target in progress messages.
func (t backlogTarget) describe(projName string, fromPathFlag bool) string {
	if projName == "" || fromPathFlag {
		return t.dir
	}
	return fmt.Sprintf("%s (%s %s)", t.dir, projName, t.repo)
}

// resolveBacklogTargets picks the repos to scan: --path scans that folder;
// --repo scans that repo of the project; with neither, every repo the project
// has a path for. Without a project (or one with no paths), the current folder.
func resolveBacklogTargets(pathFlag, repoFlag string, proj *config.ProjectConfig, projName string) ([]backlogTarget, error) {
	if repoFlag != "" && repoFlag != "backend" && repoFlag != "frontend" {
		return nil, fmt.Errorf("--repo debe ser backend o frontend, no %q", repoFlag)
	}
	repo := repoFlag
	if repo == "" {
		repo = "backend"
	}
	if pathFlag != "" {
		return []backlogTarget{{repo: repo, dir: pathFlag}}, nil
	}

	var targets []backlogTarget
	if proj != nil {
		if repoFlag != "frontend" && proj.BackendPath != "" {
			targets = append(targets, backlogTarget{repo: "backend", dir: proj.BackendPath})
		}
		if repoFlag != "backend" && proj.FrontendPath != "" {
			targets = append(targets, backlogTarget{repo: "frontend", dir: proj.FrontendPath})
		}
	}
	if len(targets) > 0 {
		return targets, nil
	}

	// Asking for a repo the project does not configure is an error: falling
	// back to the current folder would scan the wrong repo under its name.
	if proj != nil && repoFlag != "" {
		return nil, fmt.Errorf("el proyecto %s no tiene %s_path configurado. Usa --path o: gtt project add", projName, repoFlag)
	}
	return []backlogTarget{{repo: repo, dir: "."}}, nil
}

// validateBacklogOutput rejects a single file name for -o when several repos
// are scanned: each repo needs its own file, so -o must then be a folder.
func validateBacklogOutput(outputFlag string, targets int) error {
	if targets > 1 && strings.EqualFold(filepath.Ext(outputFlag), ".json") {
		return fmt.Errorf("con varios repos -o recibe una carpeta, no un archivo (%s). Usa -r para analizar uno solo", outputFlag)
	}
	return nil
}

// backlogOutputPath is where -o saves one repo's report: the given file when a
// single repo is scanned, <folder>/<name> when several are, and "" (the default
// location) when -o has no value.
func backlogOutputPath(outputFlag string, targets int, name string) string {
	if outputFlag == "" || targets == 1 {
		return outputFlag
	}
	return filepath.Join(outputFlag, name)
}

// reportBacklogFailure tells the user a repo could not be scanned. With --json
// it goes to stderr so stdout stays valid JSON.
func reportBacklogFailure(asJSON bool, repo string, err error) {
	if asJSON {
		fmt.Fprintf(os.Stderr, "No se pudo analizar %s: %v\n", repo, err)
		return
	}
	errLine(fmt.Sprintf("No se pudo analizar %s: %v", repo, err))
	fmt.Println()
}

// backlogReportName is the default file name of a saved report:
// <project>-<repo>.json, or <folder>-<repo>.json when no project is used.
func backlogReportName(projName, dir, repoName string) string {
	base := projName
	if base == "" {
		if abs, err := filepath.Abs(dir); err == nil {
			base = filepath.Base(abs)
		}
	}
	return base + "-" + repoName + ".json"
}

// saveBacklogReport writes the report as JSON. gtt writes the file itself
// instead of relying on shell redirection: in Windows PowerShell 5.1 `>`
// saves UTF-16, which other tools cannot read as JSON. With no path the file
// goes next to config.yaml (~/.config/gtt/backlog/), never to a synced
// folder; it holds internal paths and commit messages, so it is private (0600).
func saveBacklogReport(rep *backlog.Report, path, defaultName string) (string, error) {
	if path == "" {
		cfgPath, err := config.ConfigPath()
		if err != nil {
			return "", fmt.Errorf("no se pudo resolver la carpeta de configuración: %w", err)
		}
		path = filepath.Join(filepath.Dir(cfgPath), "backlog", defaultName)
	}
	path = filepath.Clean(path)

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("no se pudo crear la carpeta de %s: %w", path, err)
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return "", fmt.Errorf("no se pudo guardar el reporte en %s: %w", path, err)
	}
	return path, nil
}

// parseBacklogLimit reads --limit: empty means the default, 0 means all.
func parseBacklogLimit(v string) (int, error) {
	if v == "" {
		return defaultBacklogLimit, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("--limit debe ser un número mayor o igual a 0, no %q", v)
	}
	return n, nil
}

var sinceShorthand = regexp.MustCompile(`^(\d+)([dwm])$`)

// normalizeSince accepts the shorthands 90d, 12w and 3m on top of anything
// `git log --since` understands ("2026-07-01", "90 days ago").
func normalizeSince(v string) string {
	m := sinceShorthand.FindStringSubmatch(v)
	if m == nil {
		return v
	}
	unit := map[string]string{"d": "days", "w": "weeks", "m": "months"}[m[2]]
	return m[1] + " " + unit + " ago"
}

// backlogOptions layers the --since flag over the project's backlog config;
// whatever stays empty takes the generic default inside internal/backlog.
func backlogOptions(rc *config.BacklogRepoConfig, sinceFlag string) backlog.Options {
	var o backlog.Options
	if rc != nil {
		o = backlog.Options{
			Since:       rc.Since,
			Extensions:  rc.Extensions,
			Exclude:     rc.Exclude,
			MaxLines:    rc.MaxLines,
			MinCommits:  rc.MinCommits,
			FixKeywords: rc.FixKeywords,
			ClassGlobs:  rc.ClassGlobs,
			TestsDir:    rc.TestsDir,
			TestGlobs:   rc.TestGlobs,
		}
	}
	if sinceFlag != "" {
		o.Since = sinceFlag
	}
	o.Since = normalizeSince(o.Since)
	return o
}

func severityColor(s backlog.Severity) string {
	switch s {
	case backlog.SeverityHigh:
		return clRed
	case backlog.SeverityMedium:
		return clYellow
	default:
		return clCyan
	}
}

func printBacklogReport(rep *backlog.Report) {
	for _, s := range rep.Skipped {
		warnLine(fmt.Sprintf("Detector %s omitido: %s", s.Detector, s.Reason))
	}

	if len(rep.Findings) == 0 {
		okLine("Sin hallazgos en el período " + rep.Since)
		return
	}
	fmt.Println()

	points := 0
	for i, f := range rep.Findings {
		points += f.Points
		fmt.Printf("%2d. %s %s  %s %s\n",
			i+1,
			clr(severityColor(f.Severity)+clBold, fmt.Sprintf("[%-5s]", f.Severity)),
			clr(clBold, fmt.Sprintf("%d pts", f.Points)),
			f.Title,
			clr(clCyan, "("+string(f.Kind)+" · "+f.ID+")"))
		// Paths stay slash-separated in the report (stable IDs across OSes);
		// on screen they use the separator of the system gtt runs on.
		for _, file := range f.Files {
			fmt.Printf("      %s\n", filepath.FromSlash(file))
		}
		for _, e := range f.Evidence {
			fmt.Printf("      · %s\n", e)
		}
	}

	fmt.Println()
	fmt.Printf("Mostrando %d de %d hallazgos · %s sugeridos en total\n", len(rep.Findings), rep.Total, clr(clBold, fmt.Sprintf("%d pts", points)))
	if len(rep.Findings) < rep.Total {
		fmt.Println("Usa " + clBold + "--limit 0" + clReset + " para verlos todos.")
	}
}

func printBacklogUsage() {
	h := func(s string) { fmt.Print(clBold + s + clReset + "\n") }

	fmt.Printf(clCyan + clBold + "gtt backlog scan" + clReset + " — Detecta deuda técnica y la propone como tareas\n\n")
	fmt.Print("  Revisa el historial de git y el código de tus repos y lista lo que conviene\n")
	fmt.Print("  atacar: archivos que cambian demasiado, clases sin tests, TODO/FIXME pendientes\n")
	fmt.Print("  y archivos enormes. Cada hallazgo trae prioridad y puntos sugeridos.\n")
	fmt.Print("  Solo lee: no modifica nada, no usa internet (salvo --deps) y no necesita gtt init.\n\n")

	h("Uso:")
	fmt.Print("  gtt backlog scan [flags]\n\n")

	h("Qué analiza (se puede ejecutar desde cualquier carpeta):")
	fmt.Print("  1. El proyecto de -p, o el default_project de tu config.yaml\n")
	fmt.Print("  2. Todos sus repos con ruta: backend_path y frontend_path (con -r, solo uno)\n")
	fmt.Print("  3. Con --path, esa carpeta en lugar del proyecto\n")
	fmt.Print("  4. Sin proyecto ni --path, la carpeta actual\n\n")

	h("Flags:")
	fmt.Print("  -p, --project   Proyecto del config.yaml                     (por defecto: default_project)\n")
	fmt.Print("  -r, --repo      backend o frontend: analiza solo ese repo    (por defecto: todos)\n")
	fmt.Print("      --path      Carpeta a analizar, aunque no sea un proyecto\n")
	fmt.Print("      --since     Historial a mirar: 30d, 12w, 3m o una fecha  (por defecto: 90d)\n")
	fmt.Print("  -l, --limit     Hallazgos a mostrar por repo; 0 = todos      (por defecto: 10)\n")
	fmt.Print("  -o, --output    Guarda el reporte completo en JSON, un archivo por repo\n")
	fmt.Print("                  Sin ruta: " + defaultBacklogDirHint() + "<proyecto>-<repo>.json\n")
	fmt.Print("      --json      Imprime JSON en vez de la tabla (para otras herramientas)\n")
	fmt.Print("      --deps      Revisa paquetes vulnerables con composer audit (usa internet)\n")
	fmt.Print("  -h, --help      Muestra esta ayuda\n\n")

	h("Qué detecta:")
	fmt.Print("  hotspot         Archivos con 10+ commits en el período. Alta prioridad si además son grandes\n")
	fmt.Print("  sin-test        Clases que no aparecen en ningún test (requiere class_globs, ver abajo)\n")
	fmt.Print("  marcador        TODO / FIXME / HACK pendientes en el código\n")
	fmt.Print("  archivo-grande  Archivos de más de 800 líneas que casi no cambian (prioridad baja)\n")
	fmt.Print("  dependencia     Paquetes con vulnerabilidades reportadas (solo con --deps)\n\n")

	h("Configuración (opcional):")
	fmt.Print("  Sin configurar nada funcionan todos los detectores menos sin-test. Para activarlo,\n")
	fmt.Print("  agrega dentro de tu proyecto en " + configPathHint() + ":\n\n")
	fmt.Print(clCyan + "    backlog:\n")
	fmt.Print("      backend:\n")
	fmt.Print("        class_globs: [\"app/**/Services/**/*.php\"]   # archivos que deberían tener test\n")
	fmt.Print("      frontend:\n")
	fmt.Print("        class_globs: [\"src/app/**/*.service.ts\"]" + clReset + "\n\n")
	fmt.Print("  También acepta: since, min_commits, max_lines, exclude, extensions, tests_dir y\n")
	fmt.Print("  test_globs. Detalle en la guía: " + backlogGuideURL + "\n\n")

	h("Ejemplos:")
	fmt.Print("  gtt backlog scan                           # backend y frontend del proyecto por defecto\n")
	fmt.Print("  gtt backlog scan -o                        # lo mismo y guarda el reporte completo\n")
	fmt.Print("  gtt backlog scan -r frontend --since 30d   # solo el frontend, último mes\n")
	fmt.Print("  gtt backlog scan -l 0                      # todos los hallazgos, no solo el top 10\n")
	fmt.Print("  gtt backlog scan --path " + exampleRepoPath() + "    # un repo que no está en tus proyectos\n")
	fmt.Print("  gtt backlog scan -r backend -o " + exampleReportPath() + "   # un solo repo, a un archivo\n")
}

// backlogGuideURL points to the full `backlog scan` section of the user guide.
const backlogGuideURL = "https://github.com/geomark27/deploy-doc/blob/main/docs/guia-de-usuario.md#backlog-scan"

// defaultBacklogDirHint shows where -o saves a report without a path, written
// the way the user's system writes paths.
func defaultBacklogDirHint() string {
	if runtime.GOOS == "windows" {
		return `%USERPROFILE%\.config\gtt\backlog\`
	}
	return "~/.config/gtt/backlog/"
}

// exampleRepoPath is a placeholder repo path in the user's system style.
func exampleRepoPath() string {
	if runtime.GOOS == "windows" {
		return `C:\repos\mi-api`
	}
	return "~/repos/mi-api"
}

// exampleReportPath is a placeholder report path in the user's system style.
func exampleReportPath() string {
	if runtime.GOOS == "windows" {
		return `C:\reportes\backlog.json`
	}
	return "~/reportes/backlog.json"
}
