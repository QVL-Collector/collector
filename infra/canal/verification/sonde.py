"""Usage : sonde.py droits|ecartes|reseau <hote:port:ouvert|ferme>...
Sortie : ac|cas|identite|attendu|obtenu|atteint ; mots de passe lus dans MDP_<IDENTITE>."""

import os
import socket
import sys
import time
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
LIMITE_LIVRAISONS = 5

ECRIT = {
    "marche": [("controle.demandes", "v1", "controle.demandes", "controle"),
               ("pdm.evenements", "publication.decidee.v1", "interactions.publication-decidee", "interactions")],
    "controle": [("controle.resultats", "v1", "pdm.resultats-controle", "marche")],
}
LIT = {
    "marche": "pdm.resultats-controle",
    "controle": "controle.demandes",
    "interactions": "interactions.publication-decidee",
}
INTERDIT_EN_ECRITURE = {
    "marche": [("controle.resultats", "v1"), ("pdm.evenements", "publication.annulee.v1")],
    "controle": [("controle.demandes", "v1"), ("pdm.evenements", "publication.decidee.v1")],
    "interactions": [("pdm.evenements", "publication.decidee.v1"), ("controle.resultats", "v1")],
    "keda": [("controle.demandes", "v1")],
}
INTERDIT_EN_LECTURE = {
    "marche": ["controle.demandes", "interactions.publication-decidee"],
    "controle": ["pdm.resultats-controle", "controle.demandes.ecartes"],
    "interactions": ["controle.demandes", "pdm.resultats-controle"],
    "keda": ["controle.demandes"],
}
USURPE = {"marche": "controle", "controle": "marche"}
FILE_D_UN_AUTRE = {
    "marche": "controle.demandes",
    "controle": "interactions.publication-decidee",
    "interactions": "pdm.resultats-controle",
    "keda": "controle.demandes",
}


def joignable(hote, port, delai):
    fin = time.monotonic() + delai
    while True:
        try:
            socket.create_connection((hote, port), timeout=3).close()
            return True
        except OSError:
            if time.monotonic() >= fin:
                return False
            time.sleep(1)


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
    except Exception as e:
        return "erreur %s" % type(e).__name__
    finally:
        if conn is not None and conn.is_open:
            conn.close()


def publier(echange, cle, user_id=None, message_id=None):
    def op(canal):
        proprietes = pika.BasicProperties(
            delivery_mode=2,
            message_id=message_id or str(uuid.uuid4()),
            user_id=user_id,
            content_type="application/json",
        )
        canal.basic_publish(echange, cle, b'{"sonde":"us-011"}', proprietes, mandatory=True)

    return op


def consommer(file):
    return lambda canal: canal.basic_consume(file, lambda *a: None)


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


def cas(ac, nom, identite, attendu, obtenu):
    atteint = "oui" if obtenu == attendu or obtenu.startswith(attendu + " ") else "non"
    resultats.append(atteint)
    print("|".join([ac, nom, identite, attendu, obtenu, atteint]), flush=True)


def lire_marque(file, marque, producteur):
    def op(canal):
        for _ in range(50):
            methode, proprietes, _corps = canal.basic_get(file)
            if methode is None:
                break
            if proprietes.message_id == marque:
                canal.basic_ack(methode.delivery_tag)
                if proprietes.user_id != producteur:
                    raise RuntimeError("user-id %s" % proprietes.user_id)
                return
            canal.basic_nack(methode.delivery_tag, requeue=True)
        raise LookupError("message absent")

    return op


def droits():
    joignable(HOTE, PORT, 30)
    for producteur, sujets in ECRIT.items():
        for echange, cle, file, lecteur in sujets:
            marque = "temoin-" + str(uuid.uuid4())
            cas("AC2", "publication:%s/%s" % (echange, cle), producteur, "OK",
                tenter(publier(echange, cle, user_id=producteur, message_id=marque), producteur))
            obtenu = tenter(lire_marque(file, marque, producteur), lecteur)
            cas("AC2", "livraison:%s" % file, lecteur, "OK", obtenu)

    for identite, sujets in INTERDIT_EN_ECRITURE.items():
        for echange, cle in sujets:
            cas("AC4", "publication-interdite:%s/%s" % (echange, cle), identite, "403",
                tenter(publier(echange, cle, user_id=identite), identite))
    for identite, files in INTERDIT_EN_LECTURE.items():
        for file in files:
            cas("AC4", "lecture-interdite:%s" % file, identite, "403", tenter(consommer(file), identite))
    for identite, autre in USURPE.items():
        echange, cle, _f, _l = ECRIT[identite][0]
        cas("AC4", "user-id-usurpe:%s" % autre, identite, "406",
            tenter(publier(echange, cle, user_id=autre), identite))

    for identite in list(LIT) + ["keda"]:
        propre = LIT.get(identite, "controle.demandes")
        cas("AC5", "declaration-file", identite, "403", tenter(declarer_file("intrus." + identite), identite))
        cas("AC5", "declaration-file:" + propre, identite, "403", tenter(declarer_file(propre), identite))
        cas("AC5", "declaration-echange", identite, "403", tenter(declarer_echange("intrus." + identite), identite))
        cas("AC5", "abonnement:%s->pdm.evenements#" % propre, identite, "403",
            tenter(lier(propre, "pdm.evenements", "#"), identite))
        cas("AC5", "suppression:" + propre, identite, "403", tenter(supprimer(propre), identite))
        autre = FILE_D_UN_AUTRE[identite]
        cas("AC5", "purge-file-autre:" + autre, identite, "403", tenter(purger(autre), identite))
        cas("AC5", "suppression-file-autre:" + autre, identite, "403", tenter(supprimer(autre), identite))

    try:
        connexion("guest", "guest").close()
        obtenu = "OK"
    except (ProbableAuthenticationError, ProbableAccessDeniedError):
        obtenu = "403 ACCESS_REFUSED (authentification)"
    except AMQPConnectionError as e:
        obtenu = "connexion impossible : %s" % type(e).__name__
    cas("MC1", "connexion-guest", "guest", "403", obtenu)


def ecartes():
    joignable(HOTE, PORT, 30)
    marque = "ecarte-" + str(uuid.uuid4())
    suivant = "suivant-" + str(uuid.uuid4())

    def publier_deux(canal):
        for m in (marque, suivant):
            canal.basic_publish("controle.demandes", "v1", b'{"sonde":"mc8"}',
                                pika.BasicProperties(delivery_mode=2, message_id=m, user_id="marche"))

    cas("MC8", "publication:rejete-et-suivant", "marche", "OK", tenter(publier_deux, "marche"))
    livraisons = {"n": 0, "suivant": False}

    def rejeter_en_boucle(canal):
        def recu(ch, methode, proprietes, _corps):
            if proprietes.message_id == marque:
                livraisons["n"] += 1
                ch.basic_reject(methode.delivery_tag, requeue=True)
            elif proprietes.message_id == suivant:
                livraisons["suivant"] = True
                ch.basic_ack(methode.delivery_tag)
            else:
                ch.basic_reject(methode.delivery_tag, requeue=True)

        canal.basic_qos(prefetch_count=1)
        canal.basic_consume("controle.demandes", recu)
        canal.connection.call_later(10, canal.stop_consuming)
        canal.start_consuming()

    tenter(rejeter_en_boucle, "controle")
    cas("MC8", "livraisons-avant-mise-a-l-ecart", "controle", str(LIMITE_LIVRAISONS), str(livraisons["n"]))
    cas("MC8", "message-suivant-non-bloque", "controle", "oui", "oui" if livraisons["suivant"] else "non")
    trouve = {"ok": False}

    def lire_ecartes(canal):
        for _ in range(100):
            methode, proprietes, _corps = canal.basic_get("controle.demandes.ecartes")
            if methode is None:
                return
            canal.basic_ack(methode.delivery_tag)
            if proprietes.message_id == marque:
                trouve["ok"] = True
                return

    tenter(lire_ecartes, "admin-canal")
    cas("MC8", "present-dans:controle.demandes.ecartes", "admin-canal", "oui", "oui" if trouve["ok"] else "non")


def reseau(cibles):
    for cible in cibles:
        hote, port, attendu = cible.rsplit(":", 2)
        obtenu = "ouvert" if joignable(hote, int(port), 30 if attendu == "ouvert" else 0) else "ferme"
        cas("AC7", "tcp:%s:%s" % (hote, port), os.environ.get("ORIGINE", "?"), attendu, obtenu)


if __name__ == "__main__":
    scenario = sys.argv[1] if len(sys.argv) > 1 else ""
    if scenario == "droits":
        droits()
    elif scenario == "ecartes":
        ecartes()
    elif scenario == "reseau" and len(sys.argv) > 2:
        reseau(sys.argv[2:])
    else:
        sys.exit("usage : sonde.py droits|ecartes|reseau <hote:port:ouvert|ferme>...")
    sys.exit(0 if resultats and all(r == "oui" for r in resultats) else 1)
