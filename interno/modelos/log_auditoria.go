package modelos

import (
	"encoding/json"
	"time"
)

// LogAuditoria espelha a tabela logs_auditoria.
type LogAuditoria struct {
	ID              string          `json:"id"`
	IDRede          *string         `json:"id_rede,omitempty"`
	IDUsuarioAtor   *string         `json:"id_usuario_ator,omitempty"`
	TipoEvento      string          `json:"tipo_evento"`
	TipoEntidade    string          `json:"tipo_entidade"`
	IDEntidade      *string         `json:"id_entidade,omitempty"`
	DadosAnteriores json.RawMessage `json:"dados_anteriores,omitempty"`
	DadosNovos      json.RawMessage `json:"dados_novos,omitempty"`
	IPOrigem        *string         `json:"ip_origem,omitempty"`
	AgenteUsuario   *string         `json:"agente_usuario,omitempty"`
	CriadoEm        time.Time       `json:"criado_em"`
}
