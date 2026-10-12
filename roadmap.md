# teachat — roadmap

Chat de terminal para desenvolvedores e sysadmins. Minimalista: leve, sem distrações, usado do jeito que se usa qualquer ferramenta Linux.

## Princípios

- **Flags, não URLs.** O usuário nunca digita `ws://...?room=x`:
  ```bash
  teachat --room minhaempresa --password xxx --alias daniel
  ```
  O jeito como isso chega ao servidor é detalhe interno do cliente.
- **Funciona com pipes.** O teachat é uma ferramenta Unix como qualquer outra.
- **Auto-hospedável.** Uma empresa pode rodar a própria instância num servidor interno, sem tráfego indo para a internet (`--server`).
- **Minimalista.** Só entra feature que se paga.

## Estado atual (2026-10-11)

- **Servidor** (`cmd/server`): salas em memória, um mutex por sala, porta fixa `:23456`.
- **Cliente** (`cmd/client`): flags `--server`, `--room` e `--alias`.
- Ainda sem criptografia: o tráfego é `ws://` em texto aberto.
- As features entregues ficam registradas em [features.md](features.md).

## Próximos passos

Nesta ordem. Cada um é pequeno de propósito.

1. **Flag `--addr` no servidor.** Hoje a porta é fixa no código. Para auto-hospedar, ela precisa ser configurável.
2. **UI que não se atropela.** Hoje a mensagem recebida é escrita por cima da linha que está sendo digitada. Plano: `golang.org/x/term` (`term.NewTerminal`), que redesenha a linha de digitação abaixo das mensagens recebidas.
3. **Modo pipe.**
   ```bash
   comandocomplicado | teachat --room minhaempresa --password xxx
   ```
   O cliente detecta que o stdin não é um terminal, envia cada linha e sai no final. Uso: mandar logs e saídas de comandos para uma sala privada do time.
4. **Salas com senha + TLS, juntos.** Senha nunca deve trafegar em `ws://` aberto. TLS significa `ListenAndServeTLS` no servidor e `wss://` no cliente, com um certificado autoassinado fixado no cliente (pinning) para uso em rede local.
5. **Histórico de mensagens.** Necessário para o modo pipe ser útil quando não há ninguém na sala, e para salas persistentes. Plano: SQLite com `modernc.org/sqlite` (Go puro, então continua sendo um binário único).

## Futuro

- **Ponte com o Discord.** Assinar canais do Discord (que tenham o bot do teachat) e ler as mensagens no terminal, em vez de no app do Discord, que é pesado e tem muito ruído. Só leitura no começo.
- **Listagem de salas** (`teachat --list`) para salas públicas.

## Decisões tomadas

- Entrar numa sala que não existe cria a sala.
- A sala é apagada quando o último membro sai.
- Sem `--room`, o cliente entra na sala `geral`.
- O estado das conexões fica em RAM, não num banco de dados: ele só existe enquanto o processo está vivo. O banco entra junto com o histórico.
- `Room` fica no `package main` (`cmd/server/room.go`) até alguma outra parte além do servidor precisar dela.
- O módulo se chama `github.com/daniel-cmdline/teachat`, igual ao caminho do repositório, para os imports internos e o `go install` funcionarem.

## Perguntas em aberto

- Salas com senha vs. apagar sala vazia: se a sala some quando esvazia, qualquer um pode recriá-la com outra senha. Salas com senha devem ser persistentes?
- Modo pipe: quando o pipe termina, o cliente sai ou entra no chat?
- A senha pode ir na URL de conexão, ou deve ir num cabeçalho ou numa primeira mensagem para não aparecer em logs?

## Capacidade (teste de carga, 2026-10-11)

Servidor e clientes na mesma máquina, mensagens curtas. Mostra a ordem de grandeza, não números precisos.

| Conexões | RAM do servidor | 1 mensagem para a sala inteira |
|---|---|---|
| 1.000 | 28 MB | 5,7 ms |
| 10.000 | 183 MB | 59 ms |
| 50.000 | 875 MB | 309 ms |

Cerca de 17 KB por conexão, quase tudo pilha de goroutine e buffers do WebSocket. O tempo de broadcast cresce em linha reta com o tamanho da sala. Um cliente lento ainda atrasa a própria sala. Resolver isso exige uma fila de envio por conexão.
