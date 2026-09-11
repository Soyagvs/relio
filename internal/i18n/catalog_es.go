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
}
