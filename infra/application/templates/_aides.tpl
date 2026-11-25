{{- define "collector.etiquettes" -}}
app.kubernetes.io/name: {{ .nom }}
app.kubernetes.io/part-of: collector
app.kubernetes.io/managed-by: {{ .racine.Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .racine.Chart.Name .racine.Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end -}}

{{- define "collector.image" -}}
{{- $image := index .racine.Values.images .cle | default "" -}}
{{- if not (regexMatch "@sha256:[0-9a-f]{64}$" $image) -}}
{{- fail (printf "images.%s est obligatoire et épinglée par empreinte (@sha256:...)" .cle) -}}
{{- end -}}
{{- if not (hasPrefix .racine.Values.registreAttendu $image) -}}
{{- fail (printf "images.%s hors du registre attendu %s" .cle .racine.Values.registreAttendu) -}}
{{- end -}}
{{- $image -}}
{{- end -}}

{{- define "collector.securitePod" -}}
automountServiceAccountToken: false
securityContext:
  runAsNonRoot: true
  runAsUser: {{ .utilisateur | default 65532 }}
  runAsGroup: {{ .groupe | default 65532 }}
  seccompProfile:
    type: RuntimeDefault
{{- end -}}

{{- define "collector.securiteConteneur" -}}
securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop: [ALL]
{{- end -}}

{{- define "collector.oidc" -}}
- name: OIDC_EMETTEUR
  value: {{ printf "https://auth.%s/realms/%s" .Values.domaine .Values.identite.realm | quote }}
- name: OIDC_JEU_DE_CLES_URL
  value: {{ printf "%s/protocol/openid-connect/certs" (include "collector.keycloakInterne" .) | quote }}
{{- end -}}

{{- define "collector.keycloakInterne" -}}
{{- printf "http://keycloak.%s.svc:8080/realms/%s" .Values.identite.namespace .Values.identite.realm -}}
{{- end -}}

{{- define "collector.canal" -}}
- name: CANAL_HOTE
  value: {{ .racine.Values.canal.hote | quote }}
- name: CANAL_PORT
  value: {{ .racine.Values.canal.port | quote }}
- name: CANAL_UTILISATEUR
  value: {{ .identite | quote }}
- name: CANAL_MOT_DE_PASSE
  valueFrom:
    secretKeyRef:
      name: canal-amqp
      key: CANAL_MOT_DE_PASSE
{{- end -}}

{{- define "collector.base" -}}
- name: BASE_DE_DONNEES_URL
  valueFrom:
    secretKeyRef:
      name: {{ .secret }}
      key: BASE_DE_DONNEES_URL
- name: BASE_DE_DONNEES_TLS_EXIGE
  value: {{ .racine.Values.base.tlsExige | quote }}
{{- end -}}

{{- define "collector.volumeAutorites" -}}
- name: autorites
  configMap:
    name: {{ .Values.base.autorites }}
    optional: true
{{- end -}}

{{- define "collector.montageAutorites" -}}
- name: autorites
  mountPath: /etc/autorites-rds
  readOnly: true
{{- end -}}

{{- define "collector.sondes" -}}
startupProbe:
  httpGet: { path: /sante, port: gestion }
  periodSeconds: 2
  failureThreshold: 30
readinessProbe:
  httpGet: { path: /disponibilite, port: gestion }
  periodSeconds: 5
  failureThreshold: 2
livenessProbe:
  httpGet: { path: /sante, port: gestion }
  periodSeconds: 15
  failureThreshold: 4
{{- end -}}

{{- define "collector.hpa" -}}
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ .nom }}
  namespace: {{ .nom }}
  labels:
    {{- include "collector.etiquettes" . | nindent 4 }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ .nom }}
  minReplicas: {{ .hpa.min }}
  maxReplicas: {{ .hpa.max }}
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: {{ .hpa.cibleCpu }}
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 0
    scaleDown:
      stabilizationWindowSeconds: {{ .hpa.descenteStabilisationS }}
{{- end -}}

{{- define "collector.attenteBase" -}}
initContainers:
  - name: attente-base
    image: {{ .racine.Values.migrations.imageAttente }}
    command: ["sh", "-c", "until pg_isready -q -d \"$BASE_DE_DONNEES_URL\"; do sleep 2; done"]
    env:
      {{- include "collector.base" . | nindent 6 }}
    resources:
      {{- toYaml .racine.Values.migrations.ressources | nindent 6 }}
    {{- include "collector.securiteConteneur" . | nindent 4 }}
    volumeMounts:
      {{- include "collector.montageAutorites" . | nindent 6 }}
{{- end -}}
