package atlassian

import (
	"strings"
	"testing"
)

// sampleStorage covers the elements and macros StorageToText handles: headings,
// inline formatting, ordered and nested lists, tables, status and note macros
// and task lists.
const sampleStorage = `<h1>Resumen</h1><p>Texto <strong>clave</strong> del despliegue.</p>` +
	`<h2>Pasos</h2><ol><li>Primero</li><li>Segundo<ul><li>Detalle</li></ul></li></ol>` +
	`<h3>Tabla</h3><table><tbody><tr><th>Campo</th><th>Valor</th></tr><tr><td>Rama</td><td>main</td></tr></tbody></table>` +
	`<p>Estado: <ac:structured-macro ac:name="status"><ac:parameter ac:name="title">LISTO</ac:parameter></ac:structured-macro></p>` +
	`<ac:structured-macro ac:name="note"><ac:parameter ac:name="title">Ojo</ac:parameter><ac:rich-text-body><p>Revisar migraciones.</p></ac:rich-text-body></ac:structured-macro>` +
	`<ac:task-list><ac:task><ac:task-id>1</ac:task-id><ac:task-status>incomplete</ac:task-status><ac:task-body>Validar QA</ac:task-body></ac:task></ac:task-list>` +
	`<h4>Fin</h4>`

var (
	bar80 = strings.Repeat("=", 80)
	sep80 = strings.Repeat("-", 80)
)

// TestStorageToTextOutput pins the exact plain text produced for sampleStorage,
// so a refactor of the builder cannot change what `gtt fetch` writes.
func TestStorageToTextOutput(t *testing.T) {
	want := "\n" + bar80 + "\nRESUMEN\n" + bar80 + "\n" +
		"Texto clave del despliegue.\n" +
		"\nPasos\n" + strings.Repeat("-", 40) + "\n" +
		"1. Primero\n" +
		"2. Segundo\n   • Detalle\n" +
		"\nTabla\n" + strings.Repeat("-", 20) + "\n" +
		"\n  CAMPO | VALOR\n  Rama | main\n" +
		"Estado: [LISTO]\n" +
		"\n[NOTA: Ojo]\nRevisar migraciones.\n" +
		"  ☐ Validar QA\n" +
		"\nFin\n"

	if got := StorageToText(sampleStorage); got != want {
		t.Errorf("salida distinta.\ngot:  %q\nwant: %q", got, want)
	}
}

func TestBuildIssueTxtWithAllFields(t *testing.T) {
	page := &PageContent{
		Title:       "Titulo",
		SpaceName:   "Space",
		WebURL:      "https://example.com/p",
		CreatedDate: "2026-10-01",
		Version:     3,
		UpdatedBy:   "Usuario",
		StorageBody: "<p>Hola</p>",
	}
	want := bar80 + "\nAPP-1 - Titulo\n" + bar80 + "\n" +
		"Fuente: Confluence - Space\nURL: https://example.com/p\n" +
		"Creado: 2026-10-01 | Versión: 3\nÚltima edición: Usuario\n" +
		sep80 + "\n\nHola\n\n" +
		bar80 + "\nFIN DEL DOCUMENTO\n" + bar80 + "\n"

	if got := BuildIssueTxt("APP-1", page); got != want {
		t.Errorf("salida distinta.\ngot:  %q\nwant: %q", got, want)
	}
}

func TestBuildIssueTxtOmitsEmptyOptionalLines(t *testing.T) {
	page := &PageContent{Title: "Minimo", SpaceName: "Space", WebURL: "https://example.com/m"}
	want := bar80 + "\nAPP-2 - Minimo\n" + bar80 + "\n" +
		"Fuente: Confluence - Space\nURL: https://example.com/m\n" +
		sep80 + "\n\n" +
		bar80 + "\nFIN DEL DOCUMENTO\n" + bar80 + "\n"

	if got := BuildIssueTxt("APP-2", page); got != want {
		t.Errorf("salida distinta.\ngot:  %q\nwant: %q", got, want)
	}
}

func TestBuildIssueTxtVersionNeedsCreatedDate(t *testing.T) {
	page := &PageContent{Title: "T", Version: 5}

	if got := BuildIssueTxt("APP-3", page); strings.Contains(got, "Versión") {
		t.Errorf("sin fecha de creación no debe imprimir la versión: %q", got)
	}
}

// TestStorageToTextRegressions covers one defect each, as rendered before the
// fix and reproduced with real storage-format input.
func TestStorageToTextRegressions(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			// Whitespace between inline elements was dropped: "Rama:main".
			name: "espacio entre elementos inline",
			in:   `<p><strong>Rama:</strong> <code>main</code> y <em>dev</em></p>`,
			want: "Rama: main y dev\n",
		},
		{
			// HTML entities were left literal: "caf&eacute;", "Hola&nbsp;mundo".
			name: "entidades HTML",
			in:   `<p>Hola&nbsp;mundo &amp; caf&eacute; &lt;tag&gt;</p>`,
			want: "Hola mundo & café <tag>\n",
		},
		{
			// Every task was printed unchecked, even the completed ones.
			name: "tareas completadas",
			in: `<ac:task-list>` +
				`<ac:task><ac:task-id>1</ac:task-id><ac:task-status>complete</ac:task-status><ac:task-body>Hecho</ac:task-body></ac:task>` +
				`<ac:task><ac:task-id>2</ac:task-id><ac:task-status>incomplete</ac:task-status><ac:task-body>Pendiente</ac:task-body></ac:task>` +
				`</ac:task-list>`,
			want: "  ☑ Hecho\n  ☐ Pendiente\n",
		},
		{
			// A macro inside a panel reset the shared macro state and the panel
			// lost its "[INFO: ...]" header.
			name: "macro anidada en panel",
			in: `<ac:structured-macro ac:name="info"><ac:parameter ac:name="title">Aviso</ac:parameter>` +
				`<ac:rich-text-body><p>Estado <ac:structured-macro ac:name="status"><ac:parameter ac:name="title">OK</ac:parameter></ac:structured-macro> listo</p></ac:rich-text-body>` +
				`</ac:structured-macro>`,
			want: "\n[INFO: Aviso]\nEstado [OK] listo\n",
		},
		{
			// The sublist was glued to its parent item: "• Uno   • Uno.a".
			name: "lista anidada",
			in:   `<ul><li>Uno<ul><li>Uno.a</li><li>Uno.b</li></ul></li><li>Dos</li></ul>`,
			want: "• Uno\n   • Uno.a\n   • Uno.b\n• Dos\n",
		},
		{
			// An unclosed <br> swallowed the rest of the document.
			name: "br sin cerrar",
			in:   `<p>Linea 1<br>Linea 2</p><p>Siguiente</p>`,
			want: "Linea 1\nLinea 2\nSiguiente\n",
		},
		{
			// The start attribute of an ordered list was ignored.
			name: "lista numerada con start",
			in:   `<ol start="3"><li>Tres</li><li>Cuatro</li></ol>`,
			want: "3. Tres\n4. Cuatro\n",
		},
		{
			// Indentation between block elements must not leak into the text.
			name: "indentación entre bloques",
			in:   "<p>Uno</p>\n  <p>Dos</p>\n<ul>\n  <li>A</li>\n  <li>B</li>\n</ul>",
			want: "Uno\nDos\n• A\n• B\n",
		},
	}

	for _, tc := range cases {
		if got := StorageToText(tc.in); got != tc.want {
			t.Errorf("%s:\ngot:  %q\nwant: %q", tc.name, got, tc.want)
		}
	}
}

// TestStorageToTextMarksTruncatedOutput checks that a body the parser cannot
// read to the end keeps the text read so far and says it is incomplete, instead
// of silently passing for the whole document.
func TestStorageToTextMarksTruncatedOutput(t *testing.T) {
	got := StorageToText(`<p>Antes</p><p>a</q><p>Despues</p>`)

	if !strings.HasPrefix(got, "Antes\na") || !strings.Contains(got, truncatedNotice) {
		t.Errorf("esperaba el texto leído más el aviso de truncado, obtuve: %q", got)
	}
}
