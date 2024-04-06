package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	maxGoRoutinesAllowed = 2
	waitTimeProducer     = time.Millisecond * 500
	waitTimeReader       = time.Second * 2
)

var names = []string{
	"Grogu",
	"Ahsoka Tano",
	"Luke Skywalker",
	"Bo-katan Kryze",
	"Boba Fett",
	"Moff Gideon",
	"Cara Dune",
	"Greef Karga",
	"The Armorer",
	"Fennec Shand",
}

func main() {
	log.Print("[ Thread Principal ] Comecei o processamento")
	mainCtx := context.Background()

	stopCtx, stop := signal.NotifyContext(mainCtx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)
	defer stop()

	errorGroup, _ := errgroup.WithContext(stopCtx)

	errorGroup.Go(
		func() error {
			return listenAndConsume(stopCtx)
		},
	)

	// o errorGroup.Wait will return err if any go routine in the group fails
	if err := errorGroup.Wait(); err != nil {
		log.Fatalf("[ Thread Principal ] Um erro ocorreu dentro do errorGroup: %v", err)
	}
}

func listenAndConsume(ctx context.Context) error {
	msgChan := make(chan string)
	stopCtx, cancelConsumer := context.WithCancel(ctx)
	consumerErrorGroup, consumerErrorCtx := errgroup.WithContext(stopCtx)

	consumerErrorGroup.Go(func() error {
		return produceNames(consumerErrorCtx, msgChan)
	})

	consumerErrorGroup.Go(func() error {
		return readNames(consumerErrorCtx, msgChan)
	})

	consumerErrorGroup.Go(func() error {
		// Essa go routine é a responsável por escutar problemas com o worker
		<-consumerErrorCtx.Done()
		log.Println("[Go Routine anunciadora do caos ] O consumerErrorGroup teve um erro. Avisarei ao producer e ao reader que é hora de parar.")
		cancelConsumer()

		return fmt.Errorf("[Go Routine anunciadora do caos ] Alguma coisa deu errado 🫣  | Erro original: %v", consumerErrorCtx.Err())
	})

	// Comentar essa go routine abaixo para que o programa espere infinitamente que um novo nome chegue no canal
	go func() {
		time.Sleep(time.Second * 15)
		mando := "Din Djarin"
		log.Println("[ Go Routine Mando ] Joguei no canal o nome:", mando)
		msgChan <- mando

		// Close usado apenas para demonstrar que caso o canal seja fechado, o tratamento de erros está adequado
		time.Sleep(time.Second * 2)
		log.Println("[ Go Routine Mando ] Fechando o canal 2s após adicionar o", mando, ", para dar tempo de processá-lo")
		close(msgChan)
	}()

	log.Println("[ listenAndConsume ] Esperando pra ver se alguma coisa explode")
	if err := consumerErrorGroup.Wait(); err != nil {
		log.Fatalf("[ listenAndConsume ] O consumerErrorGroup detectou um erro: %v", err)
	}

	return nil
}

// This is the our consumer
func produceNames(ctx context.Context, channel chan string) error {
	// Aqui seria a função que lê do SQS eternamente
	// Fica fazendo long polling na AWS ou reabre a conexão após N segundos de timeout

	for _, name := range names {
		select {
		case <-ctx.Done():
			log.Printf("[ produceNames ] Contexto avisou que deu ruim em algum lugar: %v", ctx.Err())
			close(channel) // Fechar o canal aqui caso não seja fechado pela routine do Mandaloriano
			return ctx.Err()
		default:
			log.Printf("[ produceNames ] Joguei no canal o nome: %s | Esperando %dms...\n", name, waitTimeProducer.Milliseconds())
			channel <- name
			time.Sleep(waitTimeProducer)
		}
	}

	log.Println("[ produceNames ] Leu tudo da lista, finalizando go routine")

	// OBS: Descomentar para forçar o fechamento do canal ao acabar de processar a lista de nomes
	// defer close(channel)
	return nil
}

func readNames(ctx context.Context, channel chan string) error {
	handlerErrGroup, _ := errgroup.WithContext(ctx)

	// Esse é o cara que controla quantas go routines podem ser lançadas ao mesmo tempo
	// Craque do jogo ⭐
	handlerErrGroup.SetLimit(maxGoRoutinesAllowed)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[ readNames ] Contexto avisou que deu ruim em algum lugar: %v", ctx.Err())
			return ctx.Err()
		case name, isOpen := <-channel:
			// OBS: Com esse trecho abaixo o worker morre quando o canal é fechado
			if !isOpen {
				log.Printf("[ readNames ] Canal fechou, finalizando processamento")
				return fmt.Errorf("o canal foi fechado, finalizando processamento")
			}
			handlerErrGroup.Go(func() error {
				log.Printf("[ readNames ] Nome lido foi: %s | Esperando %.1fs...\n", name, waitTimeReader.Seconds())
				time.Sleep(waitTimeReader)
				return nil
			})
		}
	}
}
