package middlewares

import (
	"context"

	"gaspass-servidor/interno/modelos"
)

type chaveContexto string

const (
	chaveRequestID chaveContexto = "request_id"
	chaveUsuario   chaveContexto = "usuario_sessao"
)

func ComRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, chaveRequestID, requestID)
}

func ObterRequestID(ctx context.Context) string {
	valor, _ := ctx.Value(chaveRequestID).(string)
	return valor
}

func ComUsuario(ctx context.Context, usuario *modelos.UsuarioSessao) context.Context {
	return context.WithValue(ctx, chaveUsuario, usuario)
}

func Usuario(ctx context.Context) *modelos.UsuarioSessao {
	valor, _ := ctx.Value(chaveUsuario).(*modelos.UsuarioSessao)
	return valor
}
