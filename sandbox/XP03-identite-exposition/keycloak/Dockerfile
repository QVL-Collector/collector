# Keycloak en mode optimise (D06 faisabilite) : options de build figees dans l'image,
# demarrage par "start --optimized --import-realm".
ARG KEYCLOAK_BASE_IMAGE
FROM ${KEYCLOAK_BASE_IMAGE} AS construction
ENV KC_DB=postgres \
    KC_HEALTH_ENABLED=true \
    KC_METRICS_ENABLED=false \
    KC_FEATURES_DISABLED=impersonation,kerberos,device-flow,ciba,par
COPY --chown=keycloak:keycloak mots-de-passe-interdits.txt /opt/keycloak/data/password-blacklists/mots-de-passe-interdits.txt
RUN /opt/keycloak/bin/kc.sh build

ARG KEYCLOAK_BASE_IMAGE
FROM ${KEYCLOAK_BASE_IMAGE}
COPY --from=construction /opt/keycloak/ /opt/keycloak/
USER 1000
ENTRYPOINT ["/opt/keycloak/bin/kc.sh"]
CMD ["start", "--optimized", "--import-realm"]
