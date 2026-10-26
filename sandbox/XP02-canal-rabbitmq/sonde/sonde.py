"""Sonde des droits du canal (XP02, XC2 et XC9 hors reseau).

Chaque cas ouvre sa propre connexion avec une identite de service et tente une operation.
Sortie : une ligne par cas, champs separes par '|' :
    critere|cas|identite|attendu|obtenu|atteint
Les mots de passe sont lus dans l'environnement (MDP_SVC_PDM, ...), jamais en argument.
"""

import os
import sys
import uuid

import pika
from pika.exceptions import (
    AMQPConnectionError,
    ChannelClosedByBroker,
    ProbableAccessDeniedError,
    ProbableAuthenticationError,
)

HOTE = os.environ.get("AMQP_HOTE", "rabbitmq")
PORT = int(os.environ.get("AMQP_PORT", "5672"))
VHOST = "collector"

ECRIT = {
    "svc-pdm": ("controle.demandes", "v1"),
    "svc-controle": ("controle.resultats", "v1"),
}
LIT = {
    "svc-pdm": "pdm.resultats-controle",
    "svc-controle": "controle.demandes",
    "svc-interactions": "interactions.publication-decidee",
    "svc-test": "test.publication-decidee",
}
ECHANGE_D_UN_AUTRE = {
    "svc-pdm": ("controle.resultats", "v1"),
    "svc-controle": ("controle.demandes", "v1"),
    "svc-interactions": ("pdm.evenements", "publication.decidee.v1"),
    "svc-test": ("pdm.evenements", "publication.decidee.v1"),
}
FILE_D_UN_AUTRE = {
    "svc-pdm": "controle.demandes",
    "svc-controle": "pdm.resultats-controle",
    "svc-interactions": "test.publication-decidee",
    "svc-test": "interactions.publication-decidee",
}
USURPE = {"svc-pdm": "svc-controle", "svc-controle": "svc-pdm"}


def mot_de_passe(identite):
    return os.environ["MDP_" + identite.upper().replace("-", "_")]


def connexion(identite, mdp=None):
    parametres = pika.ConnectionParameters(
        host=HOTE,
        port=PORT,
        virtual_host=VHOST,
        credentials=pika.PlainCredentials(identite, mdp if mdp is not None else mot_de_passe(identite)),
        connection_attempts=1,
        socket_timeout=5,
        blocked_connection_timeout=10,
    )
    return pika.BlockingConnection(parametres)


def tenter(operation, identite):
    """Execute operation(canal) et renvoie le code AMQP obtenu ('OK' si aucune erreur)."""
    conn = None
    try:
        conn = connexion(identite)
        canal = conn.channel()
        canal.confirm_delivery()
        operation(canal)
        return "OK"
    except ChannelClosedByBroker as e:
        return "%d %s" % (e.reply_code, e.reply_text.split(" - ")[0])
    except ProbableAccessDeniedError:
        return "403 ACCESS_REFUSED (connexion)"
    except ProbableAuthenticationError:
        return "403 ACCESS_REFUSED (authentification)"
    except AMQPConnectionError as e:
        return "connexion impossible : %s" % type(e).__name__
    finally:
        if conn is not None and conn.is_open:
            conn.close()


def publier(echange, cle, user_id=None):
    def op(canal):
        proprietes = pika.BasicProperties(
            delivery_mode=2,
            message_id=str(uuid.uuid4()),
            user_id=user_id,
            content_type="application/json",
        )
        canal.basic_publish(echange, cle, b'{"sonde":"xp02"}', proprietes)

    return op


def consommer(file):
    def op(canal):
        canal.basic_consume(file, lambda *a: None)

    return op


def declarer_file(nom):
    return lambda canal: canal.queue_declare(nom, durable=True, arguments={"x-queue-type": "quorum"})


def declarer_echange(nom):
    return lambda canal: canal.exchange_declare(nom, "direct", durable=True)


def lier(file, echange, cle):
    return lambda canal: canal.queue_bind(file, echange, cle)


def purger(file):
    return lambda canal: canal.queue_purge(file)


def supprimer(file):
    return lambda canal: canal.queue_delete(file)


resultats = []


def cas(critere, nom, identite, attendu, obtenu):
    atteint = "oui" if obtenu == attendu or obtenu.startswith(attendu + " ") else "non"
    resultats.append(atteint)
    print("|".join([critere, nom, identite, attendu, obtenu, atteint]), flush=True)


def refus():
    for s in LIT:
        echange, cle = ECHANGE_D_UN_AUTRE[s]
        cas("XC2", "publication-echange-autre:" + echange, s, "403", tenter(publier(echange, cle), s))
        cas("XC2", "lecture-file-autre:" + FILE_D_UN_AUTRE[s], s, "403", tenter(consommer(FILE_D_UN_AUTRE[s]), s))
        cas("XC2", "declaration-file", s, "403", tenter(declarer_file("intrus." + s), s))
        cas("XC2", "declaration-echange", s, "403", tenter(declarer_echange("intrus." + s), s))
    cas("XC2", "publication-cle-non-accordee:publication.annulee.v1", "svc-pdm", "403",
        tenter(publier("pdm.evenements", "publication.annulee.v1"), "svc-pdm"))

    for s in LIT:
        cas("XC9", "liaison-file:" + LIT[s] + "->pdm.evenements#", s, "403",
            tenter(lier(LIT[s], "pdm.evenements", "#"), s))
        cas("XC9", "purge-file-autre:" + FILE_D_UN_AUTRE[s], s, "403", tenter(purger(FILE_D_UN_AUTRE[s]), s))
        cas("XC9", "suppression-file-autre:" + FILE_D_UN_AUTRE[s], s, "403", tenter(supprimer(FILE_D_UN_AUTRE[s]), s))
    cas("XC2", "lecture-file:controle.demandes", "keda-monitoring", "403",
        tenter(consommer("controle.demandes"), "keda-monitoring"))
    cas("XC2", "publication:controle.demandes/v1", "keda-monitoring", "403",
        tenter(publier("controle.demandes", "v1"), "keda-monitoring"))
    cas("XC9", "purge-file:controle.demandes", "keda-monitoring", "403",
        tenter(purger("controle.demandes"), "keda-monitoring"))
    for s, autre in USURPE.items():
        echange, cle = ECRIT[s]
        cas("XC9", "user-id-usurpe:" + autre, s, "406", tenter(publier(echange, cle, user_id=autre), s))

    try:
        connexion("guest", "guest").close()
        obtenu = "OK"
    except (ProbableAuthenticationError, ProbableAccessDeniedError):
        obtenu = "403 ACCESS_REFUSED (authentification)"
    except AMQPConnectionError as e:
        obtenu = "connexion impossible : %s" % type(e).__name__
    cas("XC9", "connexion-guest", "guest", "403", obtenu)

    temoin()


def temoin():
    """Flux autorise de bout en bout : les refus ci-dessus ne viennent pas d'identifiants faux."""
    marque = "temoin-" + str(uuid.uuid4())

    def publier_temoin(canal):
        canal.basic_publish("controle.demandes", "v1", b'{"sonde":"temoin"}',
                            pika.BasicProperties(delivery_mode=2, message_id=marque, user_id="svc-pdm"))

    cas("TEMOIN", "publication-autorisee:controle.demandes/v1", "svc-pdm", "OK", tenter(publier_temoin, "svc-pdm"))

    def lire_temoin(canal):
        for _ in range(50):
            methode, proprietes, _corps = canal.basic_get("controle.demandes")
            if methode is None:
                break
            if proprietes.message_id == marque:
                canal.basic_ack(methode.delivery_tag)
                if proprietes.user_id != "svc-pdm":
                    raise RuntimeError("user-id inattendu")
                return
            canal.basic_nack(methode.delivery_tag, requeue=True)
        raise LookupError("message temoin absent")

    try:
        obtenu = tenter(lire_temoin, "svc-controle")
    except (LookupError, RuntimeError) as e:
        obtenu = "KO %s" % e
    cas("TEMOIN", "lecture-autorisee:controle.demandes", "svc-controle", "OK", obtenu)


def ecartes():
    """Partie broker de XC4 : un message remis en file a chaque livraison part dans .ecartes
    une fois la limite atteinte (delivery-limit 5 : 1 livraison + 5 relivraisons), sans bloquer
    le message valide publie juste apres.

    Le compteur des files quorum (x-delivery-count) n'avance qu'avec un consommateur
    (basic.consume) : un basic.get suivi d'un nack avec remise en file ne l'incremente pas
    (constat sur RabbitMQ 4.3)."""
    marque = "ecarte-" + str(uuid.uuid4())
    suivant = "suivant-" + str(uuid.uuid4())

    def publier_deux(canal):
        for m in (marque, suivant):
            canal.basic_publish("controle.demandes", "v1", b'{"sonde":"xc4"}',
                                pika.BasicProperties(delivery_mode=2, message_id=m, user_id="svc-pdm"))

    cas("XC4", "publication:message-rejete-et-suivant", "svc-pdm", "OK", tenter(publier_deux, "svc-pdm"))

    livraisons = {"n": 0, "suivant": False}

    def rejeter_en_boucle(canal):
        def recu(ch, methode, proprietes, _corps):
            if proprietes.message_id == marque:
                livraisons["n"] += 1
                ch.basic_reject(methode.delivery_tag, requeue=True)
            elif proprietes.message_id == suivant:
                livraisons["suivant"] = True
                ch.basic_ack(methode.delivery_tag)
                ch.stop_consuming()
            else:
                ch.basic_reject(methode.delivery_tag, requeue=True)

        canal.basic_qos(prefetch_count=1)
        canal.basic_consume("controle.demandes", recu)
        canal.connection.call_later(15, canal.stop_consuming)
        canal.start_consuming()

    tenter(rejeter_en_boucle, "svc-controle")
    cas("XC4", "livraisons-avant-mise-a-l-ecart", "svc-controle", "6", str(livraisons["n"]))
    cas("XC4", "message-suivant-non-bloque", "svc-controle", "oui", "oui" if livraisons["suivant"] else "non")

    trouve = {"ok": False}

    def lire_ecartes(canal):
        for _ in range(100):
            methode, proprietes, _corps = canal.basic_get("controle.demandes.ecartes")
            if methode is None:
                return
            if proprietes.message_id == marque:
                trouve["ok"] = True
                canal.basic_ack(methode.delivery_tag)
                return
            canal.basic_nack(methode.delivery_tag, requeue=True)

    tenter(lire_ecartes, "admin-canal")
    cas("XC4", "present-dans:controle.demandes.ecartes", "admin-canal", "oui", "oui" if trouve["ok"] else "non")


SCENARIOS = {"refus": refus, "ecartes": ecartes}

if __name__ == "__main__":
    if len(sys.argv) != 2 or sys.argv[1] not in SCENARIOS:
        sys.exit("usage : sonde.py refus|ecartes")
    SCENARIOS[sys.argv[1]]()
    sys.exit(0 if all(r == "oui" for r in resultats) else 1)
