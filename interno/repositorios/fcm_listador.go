package repositorios

// FCMListador lista tokens FCM de um utilizador (push).
type FCMListador interface {
	ListarTokensFCMPorUsuarioID(idUsuario string) ([]string, error)
}
