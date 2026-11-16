package identite

const (
	ImageKeycloak              = "quay.io/keycloak/keycloak:26.8.1@sha256:a78d6b79572168cb25886b80a8b5b8d4ed4390743e3eecb810538683e209dd6b"
	FichierDuRealm             = "realm-collector.json"
	Realm                      = "collector"
	ClientDuSite               = "collector-web"
	ClientTransactions         = "transactions"
	PorteeReserver             = "reserver"
	VariableSecretTransactions = "SECRET_CLIENT_TRANSACTIONS"
	RoleAdmin                  = "admin"
	RoleVendeur                = "vendeur"
	RoleAcheteur               = "acheteur"
)
