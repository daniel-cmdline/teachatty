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

## Estado atual (2026-10-12)

- **Servidor** (`cmd/server`): salas em memória, um mutex por sala, porta 7347 por padrão (`--addr` para mudar), mensagens até 1 MB. A sala `local` nasce com o servidor e nunca é apagada.
- **Cliente** (`cmd/client`): flags `--server`, `--room` e `--alias`. Tela cheia com tview (`internal/ui`): mensagens em cima, caixa de digitação embaixo.
- Modo pipe: `comando | teachat` envia cada linha para a sala, mostra no terminal e sai no fim da entrada.
- Ainda sem criptografia: o tráfego é `ws://` em texto aberto.
- As features entregues ficam registradas em [features.md](features.md).

## Próximos passos

Nesta ordem. Cada um é pequeno de propósito.

1. **Salas com senha + TLS, juntos.** Senha nunca deve trafegar em `ws://` aberto. TLS significa `ListenAndServeTLS` no servidor e `wss://` no cliente, com um certificado autoassinado fixado no cliente (pinning) para uso em rede local.
2. **Histórico de mensagens.** Necessário para o modo pipe ser útil quando não há ninguém na sala, e para salas persistentes. Plano: SQLite com `modernc.org/sqlite` (Go puro, então continua sendo um binário único).

## Futuro

- **Ponte com o Discord.** Assinar canais do Discord (que tenham o bot do teachat) e ler as mensagens no terminal, em vez de no app do Discord, que é pesado e tem muito ruído. Só leitura no começo.
- **Listagem de salas** (`teachat --list`) para salas públicas.
- **Mensagens estruturadas** (`{"alias": ..., "text": ...}` em vez de `alias: texto`). Tira o parsing frágil do cliente e abre espaço para avisos de entrada/saída, histórico e cores escolhidas.
- **Arquivo de configuração**, para não depender de flags (e para o empacotamento em apt/dnf). Já decidido: `/etc/teachat/teachat.conf` (padrões da máquina, instalado pelo pacote) e `~/.config/teachat/config` (preferências de cada um). Precedência: padrões do código → `/etc` → arquivo do usuário → flags. Formato `chave = valor` com comentários `#`, sem dependência. Cores aceitam nomes e hex (o tcell já entende os dois); valor inválido dá erro com arquivo e linha. Senha nunca em `/etc`, só no arquivo do usuário com permissão 0600. Vale quando vier o empacotamento, a senha ou mais opções no servidor; até lá, um `alias` do shell resolve.
- **UI:** agrupar mensagens seguidas da mesma pessoa (alias só na primeira), destacar menções (`@daniel`), horário nas mensagens.

## Decisões tomadas

- Entrar numa sala que não existe cria a sala.
- A sala é apagada quando o último membro sai, menos a sala padrão.
- A sala padrão é a `local`: nasce com o servidor, nunca é apagada, e é para onde vai quem não passa `--room` (no cliente) ou não manda sala na URL (no servidor). Assim todo mundo se encontra sem combinar nada. Antes era `geral`, criada e apagada como as outras.
- O estado das conexões fica em RAM, não num banco de dados: ele só existe enquanto o processo está vivo. O banco entra junto com o histórico.
- `Room` fica no `package main` (`cmd/server/room.go`) até alguma outra parte além do servidor precisar dela.
- O cliente é tela cheia (tview), não inline. Inline, o chat iria para o scrollback do terminal e se perderia no meio da saída de comandos.
- Na tela: Enter envia, PgUp/PgDn rolam as mensagens, Ctrl+C sai. A própria mensagem é mostrada pelo cliente, não devolvida pelo servidor.
- Sem mouse na tela, para selecionar e copiar texto do chat continuar funcionando normalmente.
- A tela fica em `internal/ui`. O `Room` do servidor sai do `package main` depois, como mudança separada.
- Modo pipe sai quando a entrada acaba, sem nunca abrir a tela. Entrar no chat depois travaria scripts e cron (ninguém para apertar Ctrl+C), e o stdin já é o pipe, não o teclado.
- Modo pipe repete cada linha no terminal (como o `tee`), para quem roda o comando continuar vendo a saída. Uma mensagem por linha, sem agrupar: o caminho mais simples. Logs grandes inundam a sala; agrupar fica para depois, se incomodar.
- Porta padrão 7347 (antes 23456).
- `-h` nos dois binários, com os padrões lidos das próprias flags para a ajuda nunca ficar desatualizada. Tudo que o programa mostra no terminal (ajuda, logs, erros, mensagens de início, saída dos testes) é em inglês, por ser mais universal. Em português, por enquanto: comentários do código, o site e a documentação (`docs/`), e os registros do projeto.
- Comando que não ativa nada (argumento solto, flag que não existe) mostra a ajuda, nos dois binários, com código 2. `help` mostra a ajuda com código 0.
- Limite de 1 MB por mensagem no servidor (`SetReadLimit`). 64 KB bastaria para quem digita, mas quebraria linhas grandes de log no modo pipe (stack traces, JSON). Acima do limite, o servidor fecha só aquela conexão (close 1009).
- `--addr` no servidor no formato do Go, `host:porta`: sem host escuta em todas as interfaces, com host só naquela.
- O módulo se chama `github.com/daniel-cmdline/teachat`, igual ao caminho do repositório, para os imports internos e o `go install` funcionarem.

## Perguntas em aberto

- Salas com senha vs. apagar sala vazia: se a sala some quando esvazia, qualquer um pode recriá-la com outra senha. Salas com senha devem ser persistentes?
- A senha pode ir na URL de conexão, ou deve ir num cabeçalho ou numa primeira mensagem para não aparecer em logs?
- O alias não é verificado: qualquer um entra como `--alias daniel`. A senha protege a sala, mas não prova quem é quem dentro dela. Precisa de identidade?

## Capacidade (teste de carga, 2026-10-11)

Servidor e clientes na mesma máquina, mensagens curtas. Mostra a ordem de grandeza, não números precisos.

| Conexões | RAM do servidor | 1 mensagem para a sala inteira |
|---|---|---|
| 1.000 | 28 MB | 5,7 ms |
| 10.000 | 183 MB | 59 ms |
| 50.000 | 875 MB | 309 ms |

Cerca de 17 KB por conexão, quase tudo pilha de goroutine e buffers do WebSocket. O tempo de broadcast cresce em linha reta com o tamanho da sala. Um cliente lento ainda atrasa a própria sala. Resolver isso exige uma fila de envio por conexão.
