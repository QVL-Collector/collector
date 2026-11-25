{{- define "canal.etiquettes" -}}
app: rabbitmq
app.kubernetes.io/name: rabbitmq
app.kubernetes.io/part-of: collector-canal
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end -}}
