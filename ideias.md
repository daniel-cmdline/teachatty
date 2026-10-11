# teachat — ideias

Chat de terminal para desenvolvedores e sysadmins. Minimalista: leve, sem distrações, usado do jeito que se usa qualquer ferramenta Linux.

## Princípios

- **Comandos com flags, não URLs.** O usuário nunca digita `ws://...?sala=x`. A interface é feita de flags:
  ```bash
  teachat --room minhaempresa --password xxx --alias daniel
  ```
  O jeito como isso chega ao servidor é detalhe interno do cliente.
- **Funciona com pipes.** O teachat é uma ferramenta Unix como qualquer outra.
- **Minimalista.** Só entra feature que se paga.

## Features

### Salas (rooms)
- Usuários podem criar salas públicas ou com senha.
- Usuários podem entrar em salas públicas ou com senha.
- A mensagem só vai para quem está na mesma sala.

### Alias
- O usuário escolhe um alias (`--alias daniel`), que aparece na frente das mensagens que ele envia:
  ```
  daniel: fala pessoal
  ```

### UI que não se atropela
- Hoje, quando chega uma mensagem enquanto você digita, ela é escrita por cima da sua linha.
- O que se quer: mensagens recebidas aparecem acima, e a linha que você está digitando continua intacta.
- Caminho provável: `golang.org/x/term` (`term.NewTerminal`).

### Modo pipe
- Mandar a saída de um comando direto para uma sala:
  ```bash
  comandocomplicado | teachat --room minhaempresa --password xxx
  ```
- Uso: registrar logs e saídas de comandos num canal privado, visível para o time.

### TLS (wss://)
- Obrigatório antes de qualquer senha trafegar e antes de o servidor ficar exposto fora da rede local.

## Futuro

### Ponte com o Discord
- Assinar canais do Discord (que tenham o bot do teachat) e receber as mensagens no terminal, em vez de no app do Discord (pesado, consome muita RAM e tem muito ruído).
- Provavelmente só leitura no começo.

## Perguntas em aberto

- Entrar numa sala que não existe cria a sala?
- A sala some quando fica vazia? Se sumir, qualquer um pode recriá-la com outra senha.
- Modo pipe e logs: se ninguém estiver na sala quando a mensagem chegar, ela se perde. Ter histórico exige o servidor guardar mensagens.
- Modo pipe: depois que o pipe termina, o cliente sai ou entra no chat?
- Sem `--room`, entra numa sala padrão (ex.: `geral`)?
- Dá para listar as salas públicas (`teachat --list`)?
