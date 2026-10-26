#!/usr/bin/env bash
# Releve d'etat du broker avant / apres chaque test (protocole XP02, "Sorties").
# Usage : PLATEFORME=compose|k3d scripts/etat.sh <libelle>
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

libelle="${1:?libelle requis (ex. avant-xc1)}"
csv_init
f="$SORTIES_DIR/$(horodatage)-$PLATEFORME-$libelle.txt"
{
  echo "### list_users"; rmq rabbitmqctl -q list_users
  echo "### list_permissions -p collector"; rmq rabbitmqctl -q list_permissions -p collector
  echo "### list_topic_permissions -p collector"; rmq rabbitmqctl -q list_topic_permissions -p collector
  echo "### list_queues -p collector"
  rmq rabbitmqctl -q list_queues -p collector name type messages messages_ready messages_unacknowledged consumers policy
  echo "### list_connections"; rmq rabbitmqctl -q list_connections user state
} >"$f"
journal "Etat ecrit dans $f"
