1): Processo principal cria um canal que transportará as mensagens lidas do SQS
2): Ter um goroutine que lê as mensagens do SQS e joga para o canal
3): Iteramos nesse canal e a partir de um errgroup, criamos go routines
4): Esse errgroup terá limite de N goroutines simultâneas

### Isso garante que sempre teremos um número n limitado de routines e criaremos elas sob demanda, caso hajam mensagens na fila

Obs: Num cenário de alta quantidade de mensagens, se usarmos o .Go do errgroup, poderemos esperar muito alguma outra go routine terminar
e desbloquear espaço do limite, estourando o tempo de visibilidade da mensagem na fila
Maaaaas, podemos usar o TryGo para tentar criar a go routine, e se este não conseguir criar a go routine por ter chegado ao limite, podemos setar a visibilidade das mensagens para 0 de forma a poderem serem consumidas outra vez

