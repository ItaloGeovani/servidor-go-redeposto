package modelos

// RedeLinkSocial linha exibida no app (título + URL); plataforma define ícone sugerido.
type RedeLinkSocial struct {
	Plataforma      string `json:"plataforma"`
	TituloExibicao string `json:"titulo_exibicao"`
	URL            string `json:"url"`
}
