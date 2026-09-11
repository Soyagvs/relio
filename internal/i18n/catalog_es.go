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
}
