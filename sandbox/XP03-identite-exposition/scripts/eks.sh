#!/usr/bin/env bash
# XP03 sur l'EKS de XP01 (protocole § C) : memes commandes que k3d.sh, plateforme eks.
# Contexte kube : KUBE_CONTEXTE (defaut collector-xp01, alias pose par up.sh de XP01).
# Base : BASE=rds (defaut, compte et base keycloak dans le RDS de XP01) ou BASE=pod (PostgreSQL du pod).
# Usage : scripts/eks.sh up | k2 | k1 [secondes] [etape] | reseau | down
PLATEFORME=eks exec "$(dirname "${BASH_SOURCE[0]}")/k3d.sh" "$@"
