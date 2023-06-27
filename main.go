package main

import (
	"context"
	"log"
	"os"
	"sync"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

var (
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
)

func init() {
	InfoLogger = log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime)
	WarningLogger = log.New(os.Stdout, "[WARNING] ", log.Ldate|log.Ltime)
	ErrorLogger = log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime)
}

func main() {
	if len(os.Args) != 4 {
		ErrorLogger.Fatal("Usage: ./clone-nostr-events ./clone-nostr-events <npub> <initial relay> <new nsec>")
	}
	var npub = os.Args[1]
	var initialRelay = os.Args[2]
	var newNSEC = os.Args[3]

	_, pk, err := nip19.Decode(npub)
	if err != nil {
		ErrorLogger.Fatal(err)
	}
	pubkey := pk.(string)

	_, sk, err := nip19.Decode(newNSEC)
	if err != nil {
		ErrorLogger.Fatal(err)
	}
	newSK := sk.(string)
	newPK, err := nostr.GetPublicKey(newSK)
	if err != nil {
		ErrorLogger.Fatal(err)
	}

	relayUrlSet := getRelayUrlSet(pubkey, initialRelay)
	InfoLogger.Println("Relay set", relayUrlSet)

	var wg sync.WaitGroup
	defer wg.Wait()

	for relayUrl := range relayUrlSet.Iterator().C {
		wg.Add(1)
		go clone(pubkey, relayUrl, newPK, newSK, &wg)
	}
}

func getRelayUrlSet(pubkey string, initialRelay string) mapset.Set[string] {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	relay, err := nostr.RelayConnect(ctx, initialRelay)
	if err != nil {
		ErrorLogger.Fatal(err)
	}

	relayUrlSet := mapset.NewSet[string]()

	sub, err := relay.Subscribe(ctx, []nostr.Filter{{
		Kinds:   []int{nostr.KindRelayListMetadata},
		Authors: []string{pubkey},
		Limit:   1,
	}})
	if err != nil {
		ErrorLogger.Fatal(err)
	}

	ev := <-sub.Events
	if ev.Kind == nostr.KindRelayListMetadata {
		for _, tag := range ev.Tags.GetAll([]string{"r"}) {
			relayUrlSet.Add(tag.Value())
		}
	}

	return relayUrlSet
}

func clone(pubkey string, relayUrl string, newPK string, newSK string, wg *sync.WaitGroup) {
	defer wg.Done()

	ctx := context.Background()

	InfoLogger.Printf("[%s]: Connecting...\n", relayUrl)
	relay, err := nostr.RelayConnect(ctx, relayUrl)
	if err != nil {
		WarningLogger.Printf("[%s]: Failed to connect\n", relayUrl)
		return
	}
	InfoLogger.Printf("[%s]: Connected\n", relayUrl)

	until := nostr.Now()

	sentCount := 0

	for {
		sub, err := relay.Subscribe(ctx, []nostr.Filter{{
			Kinds: []int{
				nostr.KindSetMetadata,
				nostr.KindTextNote,
				nostr.KindContactList,
				nostr.KindRelayListMetadata,
				nostr.KindArticle,
			},
			Authors: []string{pubkey},
			Limit:   100,
			Until:   &until,
		}})
		if err != nil {
			InfoLogger.Printf("[%s]: %s\n", relayUrl, err)
			return
		}

		go func() {
			<-sub.EndOfStoredEvents
			sub.Unsub()
		}()

		evs := make([]nostr.Event, 0)
		for ev := range sub.Events {
			evs = append(evs, *ev)
			if until > ev.CreatedAt {
				until = ev.CreatedAt - 1
			}
		}

		if len(evs) == 0 {
			InfoLogger.Printf("[%s]: sent %d events\n", relayUrl, sentCount)
			return
		}

		for _, ev := range evs {
			// skip replay
			if ev.Tags.GetFirst([]string{"e"}) != nil {
				continue
			}
			newEV := nostr.Event{
				Kind:      ev.Kind,
				PubKey:    newPK,
				CreatedAt: nostr.Now(),
				Content:   ev.Content,
				Tags:      ev.Tags,
			}
			newEV.ID = newEV.GetID()
			newEV.Sign(newSK)
			relay.Publish(ctx, newEV)
			sentCount++
		}
	}
}
