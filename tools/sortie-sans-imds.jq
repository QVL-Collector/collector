def imds: "169.254.169.254";

def en_nombre: split(".") | map(tonumber) | .[0] * 16777216 + .[1] * 65536 + .[2] * 256 + .[3];

def couvre($ip):
  if test(":") then false
  else
    (split("/") as [$adresse, $longueur]
      | pow(2; 32 - (($longueur // "32") | tonumber)) as $taille
      | (($adresse | en_nombre) / $taille | floor) == (($ip | en_nombre) / $taille | floor))
  end;

def bloc_sans_imds: ((.cidr | couvre(imds)) | not) or any((.except // [])[]; couvre(imds));

def regle_sans_imds:
  (.to | type == "array" and length > 0)
  and all(.to[]; .ipBlock == null or (.ipBlock | bloc_sans_imds));

def de_sortie: (.spec.policyTypes // (if .spec.egress then ["Ingress", "Egress"] else ["Ingress"] end)) | index("Egress") != null;

def tous_les_pods:
  ((.spec.podSelector.matchLabels // {}) | length) == 0
  and ((.spec.podSelector.matchExpressions // []) | length) == 0;

[.items[] | select(de_sortie)] as $sorties
| ($sorties | any(tous_les_pods))
  and all($sorties[] | (.spec.egress // [])[]; regle_sans_imds)
