{{- define "telegram-bridge.name" -}}
telegram-bridge
{{- end }}

{{- define "telegram-bridge.fullname" -}}
{{ include "telegram-bridge.name" . }}
{{- end }}