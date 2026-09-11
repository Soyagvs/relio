package i18n

// es is the Spanish catalog.
var es = map[MessageID]string{
	PlanCurrentVersion: "Versión actual",
	PlanDetectedChange: "Cambio detectado",
	PlanNextVersion:    "Próxima versión",
	PlanForced:         " (forzado)",
	PlanPrerelease:     " (preliminar)",
	PlanFinalize:       " (finalización)",
	PlanCommitsSince:   "%d commits desde %s",

	MenuHeadline: "%s menú",
	MenuHint:     "↑/↓ mover · 1–9 saltar · ? ayuda · g guía · enter seleccionar · q salir",

	MenuReleaseLabel:      "Lanzamiento",
	MenuReleaseDesc:       "Crea un lanzamiento — final o rc, con push y publicación opcionales",
	MenuStatusLabel:       "Estado",
	MenuStatusDesc:        "Qué queda sin publicar y la versión que sugiere",
	MenuCheckLabel:        "Verificar",
	MenuCheckDesc:         "Qué commits desde la última etiqueta son Conventional Commits",
	MenuViewReleasesLabel: "Lanzamientos",
	MenuViewReleasesDesc:  "Explora versiones, lee notas, elimina alguna",
	MenuReleaseTextLabel:  "Anuncio",
	MenuReleaseTextDesc:   "Texto de lanzamiento para copiar y pegar — elige un formato",
	MenuReleaseImageLabel: "Imagen de lanzamiento",
	MenuReleaseImageDesc:  "Guarda o comparte una tarjeta PNG del lanzamiento",
	MenuAuthLabel:         "Autenticación",
	MenuAuthDesc:          "Conexión con GitHub — estado y cómo vincularla",
	MenuSetupLabel:        "Configuración inicial",
	MenuSetupDesc:         "Crea o inspecciona .release.yaml",
	MenuGuideLabel:        "Guía",
	MenuGuideDesc:         "Recorrido paso a paso de todo el flujo",
	MenuHelpLabel:         "Ayuda",
	MenuHelpDesc:          "Todos los comandos y opciones",
	MenuExitLabel:         "Salir",
	MenuExitDesc:          "Salir de Relio",

	MenuSettingsLabel: "Configuración",
	MenuSettingsDesc:  "Preferencias de idioma y pie del historial de cambios",

	SettingsTitle:                "Configuración",
	SettingsLanguageSection:      "Idioma",
	SettingsFooterSection:        "Pie del historial de cambios",
	SettingsContributorsLabel:    "Línea de colaboradores",
	SettingsContributorsDesc:     "Añade una línea de colaboradores al pie del historial de cambios",
	SettingsCompareLinkLabel:     "Enlace de comparación",
	SettingsCompareLinkDesc:      "Añade un enlace de comparación de GitHub al pie del historial de cambios",
	SettingsFooterDisabledReason: "Abre un proyecto con .release.yaml para editar estas opciones",
	SettingsHint:                 "↑/↓ mover · espacio alternar · q volver",

	PickCancelled: "→ cancelado",
	PickHint:      "↑/↓ mover · enter seleccionar · q cancelar",

	BannerTagline:         "convierte commits en lanzamientos",
	BannerCreatedBy:       "creado por",
	BannerDevBuild:        "compilación de desarrollo",
	BannerUpdateAvailable: "▲ v%s disponible",

	PlanHooksLabel:         "ganchos",
	PlanHooksBefore:        "antes: ",
	PlanHooksAfter:         "después: ",
	PlanHooksCommandsCount: "%d comandos",
	PlanVersionFilesLabel:  "Archivos de versión",
	PlanSinceBeginning:     "el principio",

	WizardChoiceConfirm: "Crear %s",
	WizardChoicePatch:   "Cambiar a patch",
	WizardChoiceMinor:   "Cambiar a minor",
	WizardChoiceMajor:   "Cambiar a major",
	WizardChoiceCancel:  "Cancelar",
	WizardHeader:        "Lanzamiento %s",
	WizardBumpFrom:      "(%s desde %s)",
	WizardConfirmed:     "→ confirmado %s",
	WizardCancelled:     "→ cancelado",
	WizardHint:          "↑/↓ mover · enter seleccionar · y confirmar · q cancelar",

	ReleasesTitle:           "Lanzamientos",
	ReleasesEmpty:           "Aún no hay lanzamientos. Crea uno desde el menú.",
	ReleasesEmptyHint:       "q volver",
	ReleasesNoNotes:         "(sin notas para esta versión)",
	ReleasesNonePlaceholder: "(ninguno)",

	ReleasesDeleteConfirm:          "¿Eliminar %s? Esto elimina la etiqueta de git y su sección del historial de cambios.  [y/N]",
	ReleasesDeleteCancelled:        "Eliminación cancelada.",
	ReleasesDeleted:                "Eliminado %s (%s).",
	ReleasesRemovedTag:             "etiqueta",
	ReleasesRemovedTagAndChangelog: "etiqueta + sección del historial de cambios",
	ReleasesChangelogWriteError:    "etiqueta eliminada, pero historial de cambios: %v",

	ReleasesHintPrintNotesExit: "mostrar notas y salir",
	ReleasesHintDeleteRelease:  "eliminar lanzamiento",
	ReleasesHintMove:           "mover",
	ReleasesHintShowExit:       "mostrar y salir",
	ReleasesHintDelete:         "eliminar",
	ReleasesHintBack:           "volver",

	GuideStep1Title:      "Qué hace Relio",
	GuideStep1Body:       "El flujo es:  código → commit → push → relio → versión + CHANGELOG + etiqueta. No se escribe nada hasta que confirmas la vista previa, y Relio nunca hace push por su cuenta a menos que se lo pidas.",
	GuideStep1NoRepoHint: "Ejecuta esto dentro de un repositorio git para seguir los pasos de abajo.",

	GuideStep2Title: "Configura `.release.yaml`",
	GuideStep2ConfiguredBody: "Ya está configurado (proyecto: %s), así que puedes saltarte `relio init`. Este es " +
		"el propio archivo de configuración de Relio en la raíz del repositorio — no tu package.json / pyproject.toml. " +
		"Solo configuración, nunca secretos: archivo de historial de cambios, prefijo de etiqueta y, " +
		"opcionalmente, version_files y hooks.",
	GuideStep2UnconfiguredBody: "Relio necesita su propio archivo, `.release.yaml`, en la raíz del repositorio — separado " +
		"de cualquier archivo de versión que tu lenguaje ya tenga (package.json, pyproject.toml, …), y siempre se lee " +
		"desde la raíz del proyecto sin importar desde dónde ejecutes relio. Solo configuración, nunca secretos: " +
		"archivo de historial de cambios, prefijo de etiqueta y, opcionalmente, version_files (lista aquí esos " +
		"archivos de lenguaje para mantenerlos sincronizados) y hooks. Créalo con `relio init`, o escribe uno " +
		"mínimo a mano — basta con `project: <nombre>`.",

	GuideStep3Title: "Escribe Conventional Commits",
	GuideStep3Body: "`feat:` incrementa el minor; `fix:` / `perf:` / `refactor:` incrementan el patch; " +
		"`feat!:` o un pie `BREAKING CHANGE:` incrementa el major. Los commits sin tipo se ignoran para el versionado.",
	GuideStep3CheckHint: "Ejecuta `relio check` para ver cuáles de tus commits califican.",

	GuideStep4Title: "Ve qué queda pendiente",
	GuideStep4Body:  "`relio status` lista los commits sin publicar y la versión que sugieren.",

	GuideStep5Title: "Crea el lanzamiento",
	GuideStep5Body: "Ejecuta `relio` sin argumentos: obtienes una vista previa y luego un pequeño asistente " +
		"(Crear / cambiar el incremento / cancelar). Al confirmar escribe la sección de CHANGELOG.md, " +
		"la confirma como `chore(release): vX.Y.Z`, y crea la etiqueta anotada — nada se escribe antes de confirmar.\n" +
		"`relio --rc` genera un release candidate que puedes iterar; volver a ejecutar `relio` sobre un rc lo finaliza.",

	GuideStep6Title: "Publícalo",
	GuideStep6Body: "Haz push con `git push --follow-tags`. O usa `relio --publish` para hacer push y crear el " +
		"GitHub Release con las notas del historial de cambios como cuerpo — eso necesita un token de GitHub " +
		"(GITHUB_TOKEN / GH_TOKEN / `gh auth login`).",
	GuideStep6NoTokenHint: "Todavía no hay un token de GitHub configurado.",

	GuideStep7Title: "Extras opcionales",
	GuideStep7Body: "`relio post` imprime texto de anuncio para redes sociales. `relio image` genera una tarjeta " +
		"PNG del lanzamiento. `.release.yaml` `version_files:` escribe la nueva versión en package.json / " +
		"pyproject.toml / …. `.release.yaml` `release.hooks.before` / `.after` ejecutan comandos de shell " +
		"alrededor del lanzamiento.",

	GuideStep8Title: "Ya estás listo",
	GuideStep8Body: "Camino feliz:  %s.\n" +
		"Consulta `relio help` para ver todos los comandos y opciones, y el README para la referencia completa de `.release.yaml`.",

	GuideActionRunInit:   "ejecutar relio init ahora",
	GuideActionRunCheck:  "ejecutar relio check ahora",
	GuideActionRunStatus: "ejecutar relio status ahora",

	GuideHappyPath:   "relio init → escribe commits feat:/fix: → relio status → relio → git push --follow-tags",
	GuidePlainHeader: "Relio — guía",
	GuidePlainFooter: "El camino feliz:  %s",

	GuideStepCounter:      "Paso %d de %d",
	GuideFooterWithAction: "[y] %s · enter saltar · ← atrás · q salir",
	GuideFooterNoAction:   "enter continuar · ← atrás · q salir",

	RootShort: "Convierte código terminado en un lanzamiento publicado",
	RootLong: "  Lee la actividad de git del repositorio y la convierte en una versión,\n" +
		"  un historial de cambios y una etiqueta — en un solo comando, con una vista\n" +
		"  previa antes de escribir nada.\n\n" +
		"  Ejecuta `relio` solo para el menú interactivo. Usa los\n" +
		"  subcomandos de abajo para configuración, extras y scripting.",

	FlagDirUsage:            "ejecuta como si relio se hubiera iniciado en `path`",
	FlagNoHashUsage:         "oculta los hashes de commit en las notas de lanzamiento",
	FlagPatchUsage:          "fuerza un incremento PATCH",
	FlagMinorUsage:          "fuerza un incremento MINOR",
	FlagMajorUsage:          "fuerza un incremento MAJOR",
	FlagYesUsage:            "omite el menú interactivo y la confirmación",
	FlagNoChangelogUsage:    "no modifica el archivo de historial de cambios",
	FlagNoTagUsage:          "no crea la etiqueta de git",
	FlagNoVersionFilesUsage: "no actualiza los archivos listados en version_files",
	FlagPublishUsage:        "hace push y crea el GitHub Release después de etiquetar",
	FlagRCUsage:             "genera un release candidate (vX.Y.Z-rc.N) en vez de la versión final",
	FlagNoHooksUsage:        "omite los hooks before/after de .release.yaml en esta ejecución",
	FlagEditUsage:           "abre las notas de lanzamiento generadas en tu editor antes de escribir",

	VersionInfoLine:        "%s %s (commit %s, compilado %s)\n",
	VersionShort:           "Imprime la versión de Relio",
	VersionUpdateAvailable: "▲ v%s disponible — brew upgrade relio",

	HelpSectionCommands:     "Comandos",
	HelpSectionReleaseFlags: "Opciones de lanzamiento",
	HelpSectionPostFlags:    "opciones de post",
	HelpSectionImageFlags:   "opciones de image",
	HelpSectionMenu:         "Menú",
	HelpFooter:              "Conventional Commits definen la versión: fix→patch, feat→minor, feat!/BREAKING→major.",

	HelpCmdRelioDesc:  "Crea un lanzamiento: versión + historial de cambios + etiqueta a partir de los commits desde la última etiqueta",
	HelpCmdStatusDesc: "Muestra qué queda sin publicar desde la última etiqueta y la versión que sugiere",
	HelpCmdCheckDesc:  "Lista qué commits desde la última etiqueta son Conventional Commits (--strict)",
	HelpCmdGuideDesc:  "Recorre todo el flujo de lanzamiento paso a paso",
	HelpCmdStatsDesc:  "Estadísticas públicas de descargas de Relio en GitHub (solo lectura, sin telemetría)",
	HelpCmdInitDesc:   "Crea .release.yaml en el repositorio actual (solo configuración, nunca secretos)",
	HelpCmdPostDesc:   "Imprime texto de lanzamiento para copiar y pegar en redes (texto solo en stdout)",
	HelpCmdImageDesc:  "Crea una imagen de tarjeta de lanzamiento — guárdala, súbela para obtener un enlace, o ambas cosas (--shape, --theme, --hash, --upload, --link-only)",
	HelpCmdAuthDesc:   "Inspecciona el token de GitHub que relio usará (`relio auth status`)",

	HelpFlagBumpDesc:        "Fuerza el incremento de versión en vez de inferirlo de los commits",
	HelpFlagYesDesc:         "Omite el menú y la confirmación (necesario en CI o una shell no interactiva)",
	HelpFlagNoChangelogDesc: "No modifica el archivo de historial de cambios",
	HelpFlagNoTagDesc:       "No crea la etiqueta de git",
	HelpFlagRCDesc:          "Genera un release candidate (vX.Y.Z-rc.N); ejecuta `relio` sobre un rc para finalizarlo",
	HelpFlagPublishDesc:     "Después de etiquetar, hace push de la rama y la etiqueta a origin y crea el GitHub Release",
	HelpFlagNoHashDesc:      "Oculta el hash de commit en cada línea de las notas de lanzamiento",
	HelpFlagDirDesc:         "Ejecuta como si Relio se hubiera iniciado en <path>",

	HelpPostMinimalDesc:   "Igual que el navegador de lanzamientos: proyecto, versión, fecha, commits, notas agrupadas (por defecto)",
	HelpPostSocialDesc:    "El más corto: \"Project -- Release\", versión · fecha · hora, luego líneas \"tipo  descripción\"",
	HelpPostTechnicalDesc: "Lista de viñetas concisa, para un historial de cambios o un canal de desarrollo",
	HelpPostCasualDesc:    "Tono informal: \"proj v1.4.0 is out. → …\"",
	HelpPostChangelogDesc: "La sección exacta que va en CHANGELOG.md",

	HelpImageShapeDesc:    "horizontal (1200×630) | vertical (1080×1920) | cuadrado (1080×1080)",
	HelpImageThemeDesc:    "naranja (por defecto) | verde | acento morado",
	HelpImageHashDesc:     "muestra el hash de commit en cada línea",
	HelpImageUploadDesc:   "también sube a un host temporal (litterbox 72h) e imprime un enlace + QR",
	HelpImageLinkOnlyDesc: "sube para obtener un enlace + QR sin escribir un archivo local",

	HelpMenuReleaseDesc:      "Elige final o rc y si publicar, luego ejecuta `relio`",
	HelpMenuStatusDesc:       "Qué queda sin publicar y la versión sugerida (igual que `relio status`)",
	HelpMenuCheckDesc:        "Qué commits desde la última etiqueta son Conventional Commits (igual que `relio check`)",
	HelpMenuReleasesDesc:     "Lista versiones, lee las notas de una versión, o elimina una (etiqueta de git + sección del historial de cambios)",
	HelpMenuAnnouncementDesc: "Elige un formato de post e imprime texto para copiar y pegar (igual que `relio post`)",
	HelpMenuReleaseImageDesc: "Elige un lanzamiento + forma, luego guarda la tarjeta, súbela para un enlace, o ambas cosas (igual que `relio image`)",
	HelpMenuAuthDesc:         "Conexión con GitHub — estado y cómo vincularla (ver `relio auth status`)",
	HelpMenuSetupDesc:        "Crea o inspecciona .release.yaml (igual que `relio init`)",
	HelpMenuGuideDesc:        "Recorrido paso a paso (igual que `relio guide`)",
	HelpMenuHelpDesc:         "Esta pantalla",

	StatusShort: "Muestra qué queda sin publicar y la versión que sugiere",

	StatusLabelCurrent:    "Actual",
	StatusLabelUnreleased: "Sin publicar",
	StatusLabelSuggested:  "Sugerida",
	StatusCommitsCount:    "%d commits",
	StatusNoSuggestion:    "—",

	StatusNothingToRelease:  "Nada para publicar.",
	StatusPrereleaseHint:    "en un pre-release — `relio` finaliza %s, `relio --rc` genera el siguiente rc",
	StatusUncommittedChange: "cambios sin confirmar en el árbol de trabajo",
	StatusReadyToRelease:    "Listo para publicar.",

	CheckShort:           "Revisa los commits desde la última etiqueta antes de publicar",
	CheckFlagStrictUsage: "sale con código distinto de cero cuando algún commit no es un Conventional Commit",

	CheckBaseLastTag:         "la última etiqueta",
	CheckNothingToCheck:      "Nada que revisar — no hay commits desde %s.",
	CheckCommitsNoTagYet:     "%d commits (sin etiqueta aún)",
	CheckCommitsSinceTag:     "%d commits desde %s",
	CheckConventionalCount:   "%d convencionales",
	CheckNonConventionalHead: "%d no convencionales:",
	CheckDetectedBump:        "Incremento detectado: %s  →  %s",
	CheckStrictError:         "%d commit(s) no son Conventional Commits (--strict)",

	InitShort:            "Crea un .release.yaml en el repositorio actual",
	InitLong:             "Escribe un .release.yaml con valores predeterminados razonables. El archivo contiene solo configuración — nunca secretos.",
	InitFlagProjectUsage: "nombre del proyecto (por defecto, el nombre del repo/remoto)",

	InitNotAGitRepo:   "no es un repositorio git — ejecuta `relio init` dentro de un repo",
	InitAlreadyExists: "%s ya existe en %s",
	InitConfigCreated: "%s creado",
	InitNextStepsHint: "  Revísalo, confírmalo y luego ejecuta `relio`.",
}
