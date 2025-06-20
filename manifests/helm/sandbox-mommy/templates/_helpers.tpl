{{/*
Expand the name of the chart.
*/}}
{{- define "sandbox-mommy.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "sandbox-mommy.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "sandbox-mommy.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "sandbox-mommy.labels" -}}
helm.sh/chart: {{ include "sandbox-mommy.chart" . }}
{{ include "sandbox-mommy.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "sandbox-mommy.selectorLabels" -}}
app.kubernetes.io/name: {{ include "sandbox-mommy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "sandbox-mommy.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "sandbox-mommy.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define  "sandbox-api.componentLabel" -}}
app.kubernetes.io/component: "sandbox-api"
{{- end }}

{{- define  "sandbox-controller.componentLabel" -}}
app.kubernetes.io/component: "sandbox-controller"
{{- end }}


{{- define "sandbox-api.serviceFQDN" -}}
sandbox-api.{{ .Release.Namespace | default "default" }}.svc
{{- end }}

{{- define "sandbox-controller.serviceFQDN" -}}
sandbox-controller.{{ .Release.Namespace | default "default" }}.svc
{{- end }}