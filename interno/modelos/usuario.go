package modelos

type Papel string

const (
	PapelSuperAdmin   Papel = "super_admin"
	PapelGestorRede   Papel = "gestor_rede"
	PapelGerentePosto Papel = "gerente_posto"
	PapelFrentista    Papel = "frentista"
	PapelCliente      Papel = "cliente"
)

type UsuarioSessao struct {
	IDUsuario    string `json:"id_usuario"`
	NomeCompleto string `json:"nome_completo"`
	IDRede       string `json:"id_rede"`
	Papel        Papel  `json:"papel"`
}
