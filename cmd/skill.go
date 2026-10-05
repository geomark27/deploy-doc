package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/geomark27/deploy-doc/internal/build"
	"github.com/geomark27/deploy-doc/internal/skill"
)

var skillSubcommands = map[string]func([]string) error{
	"status":  runSkillStatus,
	"install": runSkillInstall,
	"diff":    runSkillDiff,
	"reset":   runSkillReset,
}

func runSkill(args []string) error {
	if len(args) == 0 {
		return runSkillStatus(nil)
	}
	if isHelpArg(args[0]) {
		printSkillUsage()
		return nil
	}
	fn, ok := skillSubcommands[args[0]]
	if !ok {
		fmt.Fprintf(os.Stderr, clr(clRed, "Subcomando desconocido: ")+"skill %s\n\n", args[0])
		printSkillUsage()
		return fmt.Errorf("subcomando inválido")
	}
	return fn(args[1:])
}

func runSkillStatus(_ []string) error {
	st, dir, err := skill.Check(build.Version)
	if err != nil {
		return err
	}
	fmt.Printf(clBold+"Skill %s"+clReset+" — %s\n", skill.Name, dir)
	fmt.Printf("  gtt %s incluye su versión oficial\n\n", build.Version)

	switch st.State {
	case skill.NotInstalled:
		fmt.Println("  " + clr(clYellow, "⚠") + " No está instalada.")
		if !skill.ClaudeInstalled() {
			fmt.Println("    No se encontró Claude Code en este equipo; instálalo primero.")
			return nil
		}
		fmt.Println("    Instálala con: " + clr(clCyan, "gtt skill install"))
	case skill.UpToDate:
		fmt.Println("  " + clr(clGreen, "✓") + " Al día.")
	case skill.Outdated:
		fmt.Printf("  "+clr(clYellow, "⚠")+" Versión anterior (instalada por gtt %s), sin cambios locales.\n", st.Installed)
		fmt.Println("    Se actualiza sola al usar cualquier comando, o ya con: " + clr(clCyan, "gtt skill install"))
	case skill.Modified, skill.Unmanaged:
		if st.State == skill.Modified {
			fmt.Println("  " + clr(clYellow, "⚠") + " Tiene cambios locales: gtt no la actualiza para no perderlos.")
		} else {
			fmt.Println("  " + clr(clYellow, "⚠") + " Es una copia instalada a mano y distinta de la oficial: gtt no la toca.")
		}
		for _, name := range st.Changed {
			fmt.Printf("      %s\n", name)
		}
		fmt.Println()
		fmt.Println("    Ver diferencias:        " + clr(clCyan, "gtt skill diff"))
		fmt.Println("    Volver a la oficial:    " + clr(clCyan, "gtt skill reset") + "  (guarda respaldo de tus archivos)")
		fmt.Println("    Tus convenciones van en " + clr(clCyan, skill.LocalFile) + ", que gtt nunca modifica.")
	case skill.Newer:
		fmt.Printf("  "+clr(clYellow, "⚠")+" La instaló gtt %s, más nuevo que este. No se toca; actualiza gtt con "+clr(clCyan, "gtt update")+".\n", st.Installed)
	}
	return nil
}

func runSkillInstall(_ []string) error {
	if !skill.ClaudeInstalled() {
		dir, _ := skill.ClaudeDir()
		return fmt.Errorf("no se encontró Claude Code (%s). Instálalo y vuelve a intentar", dir)
	}
	st, dir, err := skill.Check(build.Version)
	if err != nil {
		return err
	}
	switch st.State {
	case skill.UpToDate:
		skill.SetDeclined(false)
		fmt.Println(clr(clGreen, "✓") + " La skill ya está al día en " + dir)
		return nil
	case skill.Modified, skill.Unmanaged:
		return fmt.Errorf("la skill instalada tiene cambios locales (%s). Revísalos con 'gtt skill diff' o reemplázala con 'gtt skill reset'", strings.Join(st.Changed, ", "))
	case skill.Newer:
		return fmt.Errorf("la skill la instaló gtt %s, más nuevo que este (%s). Actualiza gtt con 'gtt update'", st.Installed, build.Version)
	}
	if err := skill.Install(build.Version); err != nil {
		return err
	}
	skill.SetDeclined(false)
	fmt.Println(clr(clGreen, "✓") + " Skill " + skill.Name + " instalada en " + dir)
	fmt.Println("  Tus convenciones (rama, estándares, comandos de prueba) van en " + filepath.Join(dir, skill.LocalFile))
	return nil
}

func runSkillReset(_ []string) error {
	st, dir, err := skill.Check(build.Version)
	if err != nil {
		return err
	}
	if st.State == skill.Newer {
		return fmt.Errorf("la skill la instaló gtt %s, más nuevo que este (%s). Actualiza gtt con 'gtt update'", st.Installed, build.Version)
	}
	if st.State == skill.NotInstalled {
		return runSkillInstall(nil)
	}
	backups, err := skill.Backup(st.Changed)
	if err != nil {
		return fmt.Errorf("no se pudo respaldar la skill; no se modificó nada: %w", err)
	}
	if err := skill.Install(build.Version); err != nil {
		return err
	}
	fmt.Println(clr(clGreen, "✓") + " Skill " + skill.Name + " restablecida a la versión de gtt " + build.Version + " en " + dir)
	for _, b := range backups {
		fmt.Println("  Respaldo de tu versión: " + b)
	}
	return nil
}

// runSkillDiff shows the local edits against the official version with
// `git diff --no-index`, which every developer here already has.
func runSkillDiff(_ []string) error {
	st, dir, err := skill.Check(build.Version)
	if err != nil {
		return err
	}
	if st.State != skill.Modified && st.State != skill.Unmanaged {
		fmt.Println("No hay cambios locales que mostrar. Estado: " + skillStateText(st.State) + ".")
		return nil
	}

	gitPath, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git no encontrado en PATH: %w", err)
	}
	tmp, err := os.MkdirTemp("", "gtt-skill-oficial-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	official := skill.Official()
	for _, name := range st.Changed {
		officialPath := filepath.Join(tmp, name)
		if err := os.WriteFile(officialPath, official[name], 0644); err != nil {
			return err
		}
		fmt.Printf(clBold+"%s"+clReset+"  (oficial → tuya)\n", name)
		c := exec.Command(gitPath, "diff", "--no-index", "--", officialPath, filepath.Join(dir, name))
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		// Exit code 1 only means "there are differences".
		if err := c.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
				return fmt.Errorf("git diff falló: %w", err)
			}
		}
		fmt.Println()
	}
	return nil
}

func skillStateText(s skill.State) string {
	switch s {
	case skill.NotInstalled:
		return "no instalada"
	case skill.UpToDate:
		return "al día"
	case skill.Outdated:
		return "versión anterior sin cambios"
	case skill.Modified:
		return "con cambios locales"
	case skill.Unmanaged:
		return "copia manual distinta de la oficial"
	case skill.Newer:
		return "instalada por un gtt más nuevo"
	}
	return "desconocido"
}

// SyncSkill runs on startup, before the command. It keeps an untouched skill
// up to date, warns once per version when the user edited it, and offers the
// first install. It never fails the command: problems become a warning line.
func SyncSkill() {
	if !skill.ClaudeInstalled() {
		return
	}
	res, err := skill.Sync(build.Version)
	if err != nil {
		fmt.Fprintln(os.Stderr, clr(clYellow, "⚠")+" No se pudo revisar la skill "+skill.Name+": "+err.Error())
		return
	}

	switch {
	case res.Updated:
		fmt.Println(clr(clGreen, "✓") + " Skill " + skill.Name + " actualizada a la versión de gtt " + build.Version + ".\n")
	case len(res.NewFiles) > 0:
		fmt.Println(clr(clYellow, "⚠") + " La skill " + skill.Name + " tiene cambios locales: no se actualizó para no perderlos.")
		fmt.Println("  La versión oficial quedó al lado como " + skill.NewSuffix + ". Revisa con " + clr(clCyan, "gtt skill diff") +
			" o vuelve a la oficial con " + clr(clCyan, "gtt skill reset") + ".\n")
	case res.Status.State == skill.NotInstalled && !skill.Declined():
		offerSkillInstall()
	}
}

func offerSkillInstall() {
	dir, _ := skill.Dir()
	fmt.Println(clBold + "gtt incluye la skill " + skill.Name + " para Claude Code" + clReset)
	fmt.Println("  Convierte los hallazgos de 'gtt backlog scan' en tareas listas para Jira.")
	fmt.Println("  Se instala en " + dir + " y gtt la mantiene actualizada.")
	fmt.Print("¿Instalarla ahora? [S/n]: ")

	ans, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))
	if ans != "" && ans != "s" && ans != "si" && ans != "sí" {
		skill.SetDeclined(true)
		fmt.Println("  No se instaló. Cuando quieras: " + clr(clCyan, "gtt skill install") + "\n")
		return
	}
	if err := skill.Install(build.Version); err != nil {
		fmt.Fprintln(os.Stderr, clr(clYellow, "⚠")+" No se pudo instalar la skill: "+err.Error()+"\n")
		return
	}
	fmt.Println(clr(clGreen, "✓") + " Skill instalada. Tus convenciones van en " + filepath.Join(dir, skill.LocalFile) + "\n")
}

func printSkillUsage() {
	helpTitle("gtt skill", "Instala y mantiene la skill "+skill.Name+" para Claude Code")

	helpSection("Uso:")
	fmt.Print("  gtt skill [status|install|diff|reset]\n\n")

	helpSection("Subcomandos:")
	fmt.Print("  status    Estado de la skill instalada frente a la que trae gtt (por defecto)\n")
	fmt.Print("  install   Instala la skill (también si antes respondiste que no)\n")
	fmt.Print("  diff      Muestra tus cambios locales frente a la versión oficial (usa git)\n")
	fmt.Print("  reset     Vuelve a la versión oficial; guarda respaldo de tus archivos\n\n")

	helpSection("Cómo se mantiene:")
	fmt.Print("  La skill viaja dentro de gtt. Al ejecutar cualquier comando, gtt revisa la\n")
	fmt.Print("  copia instalada:\n")
	fmt.Print("    Sin tocar y de una versión anterior  → se actualiza sola\n")
	fmt.Print("    Con cambios tuyos                    → no se toca; la oficial queda al lado\n")
	fmt.Print("                                           como *" + skill.NewSuffix + " y se avisa una vez\n")
	fmt.Print("    No instalada                         → pregunta una sola vez\n\n")
	fmt.Print("  Tus convenciones (formato de rama, estándares, comandos de prueba) van en\n")
	fmt.Print("  " + clCyan + skill.LocalFile + clReset + ", en la carpeta de la skill. gtt lo crea una vez y nunca lo modifica.\n\n")

	helpSection("Dónde se instala:")
	fmt.Print("  ~/.claude/skills/" + skill.Name + "  (o $CLAUDE_CONFIG_DIR/skills/" + skill.Name + ")\n")
}
