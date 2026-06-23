{{/*
通用标签
*/}}
{{- define "redis-operator.labels" -}}
app.kubernetes.io/name: {{ include "redis-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ include "redis-operator.chart" . }}
{{- with .Values.extraLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
选择器标签
*/}}
{{- define "redis-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "redis-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Chart 名称
*/}}
{{- define "redis-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Chart 全名
*/}}
{{- define "redis-operator.fullname" -}}
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
Chart 标识
*/}}
{{- define "redis-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
ServiceAccount 名称
*/}}
{{- define "redis-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default .Values.serviceAccount.name .Values.serviceAccountName }}
{{- else }}
{{- default "default" .Values.serviceAccountName }}
{{- end }}
{{- end }}

{{/*
命名空间
*/}}
{{- define "redis-operator.namespace" -}}
{{- default .Release.Namespace .Values.namespace.name }}
{{- end }}
