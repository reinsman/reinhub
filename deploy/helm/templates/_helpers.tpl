{{/* Chart name. */}}
{{- define "examplesrv.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Release-aware resource name. */}}
{{- define "examplesrv.fullname" -}}
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

{{/* Chart label value. */}}
{{- define "examplesrv.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Common labels. */}}
{{- define "examplesrv.labels" -}}
helm.sh/chart: {{ include "examplesrv.chart" . }}
{{ include "examplesrv.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/* Selector labels must remain stable across upgrades. */}}
{{- define "examplesrv.selectorLabels" -}}
app.kubernetes.io/name: {{ include "examplesrv.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/* Container image. */}}
{{- define "examplesrv.image" -}}
{{- printf "%s:%s" .Values.image.repository (default .Chart.AppVersion .Values.image.tag) }}
{{- end }}

{{/* Secret containing conf.toml. */}}
{{- define "examplesrv.configSecretName" -}}
{{- default (printf "%s-config" (include "examplesrv.fullname" .)) .Values.config.existingSecret | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* PVC name. */}}
{{- define "examplesrv.persistenceClaimName" -}}
{{- default (printf "%s-data" (include "examplesrv.fullname" .)) .Values.persistence.existingClaim | trunc 63 | trimSuffix "-" }}
{{- end }}
