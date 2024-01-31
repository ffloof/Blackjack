package main

import (
	"fmt"
	"math/rand"
	"math"

	// Ignore these imports just for cpu profiling
	"flag"
	"os"
	"log"
	"runtime/pprof"
)

type Deck struct {
	Cards [10]int8 // A 2 3 4 5 6 7 8 9 XJQK
}

const CONVERT string = "A23456789X"

func (deck *Deck) String() string {
	str := ""
	for card, amount := range deck.Cards {
		char := string(CONVERT[card])
		for i:=0;i<int(amount);i++ {
			str += char
		}
	}
	return str
}

func (deck *Deck) Size() int {
	size := 0
	for _, n := range deck.Cards {
		size += int(n)
	}
	return size
}

func (deck *Deck) IsBlackJack() bool {
	if deck.Size() == 2 {
		if (deck.Cards[0] == 1) && (deck.Cards[9] == 1) {
			return true
		}
	}

	return false
}

func (deck *Deck) Shoes(decks int){
	for i := range(deck.Cards){
		deck.Cards[i] = int8(decks * 4)
	}
	deck.Cards[9] = int8(decks * 16)
}

func (deck *Deck) Count() int {
	count := 0

	/*
	for i, amount := range deck.Cards {
		value := i + 1
		count += value * int(amount)
	}*/

	// Above manually unrolled
	// TODO: investigate if its worth it now that conditional is removed
	count += 1 * int(deck.Cards[0])
	count += 2 * int(deck.Cards[1])
	count += 3 * int(deck.Cards[2])
	count += 4 * int(deck.Cards[3])
	count += 5 * int(deck.Cards[4])
	count += 6 * int(deck.Cards[5])
	count += 7 * int(deck.Cards[6])
	count += 8 * int(deck.Cards[7])
	count += 9 * int(deck.Cards[8])
	count += 10 * int(deck.Cards[9])
	

	if deck.Cards[0] >= 1 {
		if count <= 11 { count += 10 }
		if deck.Cards[1] >= 2 {
			if count <= 11 { count += 10 }
		}
	}

	return count
}

func (deck *Deck) Draw() int {
	chosenCard := rand.Intn(deck.Size())
	start := 0	

	for i, amount := range(deck.Cards) {
		end := start + int(amount)
		if start <= chosenCard && chosenCard < end {
			deck.Pull(i)
			return i
		}
		start = end	
	}	
	
	return -1
}

func (deck *Deck) Add(cardIndex int) *Deck {
	deck.Cards[cardIndex]++
	return deck
}

func (deck *Deck) Pull(cardIndex int) *Deck {
	deck.Cards[cardIndex]--
	return deck
}

func (deck Deck) PlayerGameTree(dealer Deck, hand Deck, firstTurn bool, playerTransposition map[Deck]float64) float64 {
	if hand.Count() > 21 {
		return -1
	}

	transpositionEV, contains := playerTransposition[hand]
	if contains {
		return transpositionEV
	}

	
	// Compute Stand
	standEV := deck.DealerGameTree(dealer, hand.Count(), hand.IsBlackJack(), map[Deck]float64{})

	// Compute Hit
	hitEV := 0.0
	size := float64(deck.Size())
	for card, amount := range deck.Cards {
		if amount == 0 { continue }
		probability := float64(amount) / size

		subtreeDeck := deck
		subtreeHand := hand

		subtreeDeck.Pull(card)
		subtreeHand.Add(card)
		
		expectation := subtreeDeck.PlayerGameTree(dealer, subtreeHand, false, playerTransposition)
		hitEV += probability * expectation
	}

	finalEV := math.Max(hitEV, standEV)

	if firstTurn {
		// Double Down
		doubleEV := 0.0
		for card, amount := range deck.Cards {
			if amount == 0 { continue }
			probability := float64(amount) / size

			subtreeDeck := deck
			subtreeHand := hand

			subtreeDeck.Pull(card)
			subtreeHand.Add(card)
			if subtreeHand.Count() > 21 { 
				doubleEV += probability * -2
			} else {
				expectation := subtreeDeck.DealerGameTree(dealer, subtreeHand.Count(), subtreeHand.IsBlackJack(), map[Deck]float64{})
				doubleEV += probability * 2 * expectation
			}
		}

		// Split
		splitEV := -1.0
		for card, amount := range hand.Cards {
			if amount == 2 {
				subtreeDeck := deck
				subtreeHand := hand

				subtreeHand.Pull(card)
				expectation := subtreeDeck.PlayerGameTree(dealer, subtreeHand, false, map[Deck]float64{})
				splitEV = 2 * expectation
				break
			}
		}

		fmt.Println("Double", doubleEV)
		fmt.Println("Hit", hitEV)
		fmt.Println("Stand", standEV)
		fmt.Println("Split", splitEV)

		finalEV = math.Max(finalEV, math.Max(splitEV, doubleEV))
	}
	
	playerTransposition[hand] = finalEV
	return finalEV
}

// Returns probability of player winning
func (deck Deck) DealerGameTree(dealer Deck, playerCount int, blackjack bool, dealerTransposition map[Deck]float64) float64 {
	dealerCount := dealer.Count()

	// Dealer goes bust
	if dealerCount > 21 {
		return 1
	}

	if dealerCount == 21 {
		if dealer.IsBlackJack() {
			if blackjack {
				return 0 // If player also gets a blackjack push
			}
			return -1
		}
	
		// If player has a blackjack assume 3:2 payout
		if blackjack {
			return 1.5
		}
	}

	// Dealer stands on 17 and up
	
	if dealerCount >= 17 {
		if dealerCount > playerCount {
			return -1
		} else if dealerCount == playerCount {
			return 0
		}
		return 1
	}

	transpositionEV, contains := dealerTransposition[dealer]
	if contains {
		return transpositionEV
	}

	EV := 0.0
	size := float64(deck.Size())
	for card, amount := range deck.Cards {
		if amount == 0 { continue }
		probability := float64(amount) / size

		subtreeDeck := deck
		subtreeDealer := dealer

		subtreeDeck.Pull(card)
		subtreeDealer.Add(card)
		
		subtreeEV := subtreeDeck.DealerGameTree(subtreeDealer, playerCount, blackjack, dealerTransposition)
		EV += probability * subtreeEV
	}

	dealerTransposition[dealer] = EV
	return EV
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main(){
	// Cpu profiling
	flag.Parse()
    if *cpuprofile != "" {
        f, err := os.Create(*cpuprofile)
        if err != nil {
            log.Fatal(err)
        }
        pprof.StartCPUProfile(f)
        defer pprof.StopCPUProfile()
	}
    
	testDeck := Deck{}
	testDeck.Shoes(4)

	playerDeck := Deck{}
	dealerDeck := Deck{}

	playerDeck.Add(3).Add(7) // Add 4 and 8
	dealerDeck.Add(9) // Add 10
	
	fmt.Println(testDeck.PlayerGameTree(dealerDeck, playerDeck, true, map[Deck]float64{}))

	return 

	// 2 through 12
	for card1:=1;card1<=7;card1++ {
		for dealerCard:=0;dealerCard<=9;dealerCard++ {
			testDeck := Deck{}
			testDeck.Shoes(1)

			playerDeck := Deck{}
			dealerDeck := Deck{}

			playerDeck.Add(1).Add(card1)
			dealerDeck.Add(dealerCard)
			
			fmt.Println("Hard", 3 + card1, "vs", string(CONVERT[dealerCard]))
			fmt.Println(testDeck.PlayerGameTree(dealerDeck, playerDeck, true, map[Deck]float64{}))
		}
	}

	for card2:=1;card2<=9;card2++ {
		for dealerCard:=0;dealerCard<=9;dealerCard++ {
			testDeck := Deck{}
			testDeck.Shoes(1)

			playerDeck := Deck{}
			dealerDeck := Deck{}

			playerDeck.Add(9).Add(card2)
			dealerDeck.Add(dealerCard)
			
			fmt.Println("Hard", 10 + card2, "vs", string(CONVERT[dealerCard]))
			fmt.Println(testDeck.PlayerGameTree(dealerDeck, playerDeck, true, map[Deck]float64{}))
		}
	}
}