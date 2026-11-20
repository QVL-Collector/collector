import socket


def handler(event, context):
    resultats = []
    for cible in event.get("cibles", []):
        hote, port = cible["hote"], int(cible["port"])
        try:
            with socket.create_connection((hote, port), timeout=4):
                etat = "ouvert"
        except socket.timeout:
            etat = "delai"
        except OSError:
            etat = "refus"
        resultats.append({"nom": cible["nom"], "etat": etat})
    return {"resultats": resultats}
