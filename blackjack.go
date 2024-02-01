package main

import (
	"fmt"
	"math/rand"

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
		for i:=0;i<int(amount);i++ {
			str += string(CONVERT[card])
		}
	}
	return str
}

// Get total number of cards in deck
func (deck *Deck) Size() int {
	size := 0
	for _, n := range deck.Cards {
		size += int(n)
	}
	return size
}

func (deck *Deck) IsBlackJack() bool {
	if deck.Size() == 2 {
		// 1 Ace and 1 High Card
		if (deck.Cards[0] == 1) && (deck.Cards[9] == 1) {
			return true
		}
	}

	return false
}

// Create new deck with n 52 cards in shoe
func Shoes(decks int) Deck {
	deck := Deck{}
	for i := range(deck.Cards){
		deck.Cards[i] = int8(decks * 4)
	}
	// There are 4 10s, Kings, Queens, Jacks per deck so 16 total
	deck.Cards[9] = int8(decks * 16)
	return deck
}

func (deck *Deck) Count() int {
	count := 0
	for i, amount := range deck.Cards {
		count += (i+1) * int(amount)
	}

	// For soft counts ace has already been counted as 1, see if it can be worth more
	if deck.Cards[0] >= 1 {
		if count <= 11 { count += 10 }
	}

	return count
}

// Draw random card weighted probability
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

// Add card to deck
func (deck *Deck) Add(cardIndex int) *Deck {
	deck.Cards[cardIndex]++
	return deck
}

// Remove card from deck
func (deck *Deck) Pull(cardIndex int) *Deck {
	deck.Cards[cardIndex]--
	return deck
}

func (deck Deck) PlayerGameTree(dealer Deck, hand Deck, firstTurn bool, playerTransposition map[Deck]float64) (float64, string) {
	if hand.Count() > 21 {
		return -1, "bust"
	}

	transpositionEV, contains := playerTransposition[hand]
	if contains {
		return transpositionEV, "?"
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
		
		expectation, _ := subtreeDeck.PlayerGameTree(dealer, subtreeHand, false, playerTransposition)
		hitEV += probability * expectation
	}

	finalEV := standEV
	finalAction := "stand"
	
	if hitEV > finalEV {
		finalEV = hitEV
		finalAction = "hit"
	}

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
				expectation, _ := subtreeDeck.PlayerGameTree(dealer, subtreeHand, false, map[Deck]float64{})
				splitEV = 2 * expectation
				break
			}
		}

		

		if splitEV > finalEV {
			finalEV = splitEV
			finalAction = "split"
		}

		if doubleEV > finalEV {
			finalEV = doubleEV
			finalAction = "double"
		}

		/*
		fmt.Println("Double", doubleEV)
		fmt.Println("Hit", hitEV)
		fmt.Println("Stand", standEV)
		fmt.Println("Split", splitEV)
		fmt.Println(">", finalAction, finalEV)
		*/
		
	}
	
	playerTransposition[hand] = finalEV
	return finalEV, finalAction
}

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

func computeHand(playerCard1, playerCard2, dealerCard int){
	testDeck := Shoes(6)

	playerDeck := Deck{}
	dealerDeck := Deck{}

	playerDeck.Add(playerCard1).Add(playerCard2)
	dealerDeck.Add(dealerCard)

	fmt.Println(playerDeck.Count(), " vs ", dealerDeck.String())
	
	fmt.Println(testDeck.PlayerGameTree(dealerDeck, playerDeck, true, map[Deck]float64{}))
	fmt.Println("")
}

func computeBasicStrategy(){
	for j:=0;j<=9;j++{
		// 4 through 11
		for i:=1;i<=8;i++{
			computeHand(1,i,j)
		}

		// 12 through 20
		for i:=2;i<=9;i++{
			computeHand(9,i,j)
		}


		//for i:=2;i<=9;i++{
		//	computeHand(9,i,j)
		//}
	}
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
    
	computeBasicStrategy()
}