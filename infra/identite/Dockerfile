ARG KEYCLOAK_BASE_IMAGE=quay.io/keycloak/keycloak:26.8.1@sha256:a78d6b79572168cb25886b80a8b5b8d4ed4390743e3eecb810538683e209dd6b

FROM ${KEYCLOAK_BASE_IMAGE} AS construction
ENV KC_DB=postgres \
    KC_HEALTH_ENABLED=true \
    KC_METRICS_ENABLED=false \
    KC_FEATURES_DISABLED=impersonation,kerberos,device-flow,ciba,par
RUN /opt/keycloak/bin/kc.sh build

FROM ${KEYCLOAK_BASE_IMAGE}
COPY --from=construction /opt/keycloak/ /opt/keycloak/
COPY --chown=1000:0 infra/identite/realm-collector.json /opt/keycloak/data/import/realm-collector.json
USER 1000
ENTRYPOINT ["/opt/keycloak/bin/kc.sh"]
CMD ["start", "--optimized", "--import-realm"]
