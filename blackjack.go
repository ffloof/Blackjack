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

const doubleAfterSplit bool = true
const blackjackPayout float64 = 1.5





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

func (deck *Deck) Value() int {
	value := 0
	for i, amount := range deck.Cards {
		value += (i+1) * int(amount)
	}

	// For soft values ace has already been valueed as 1, see if it can be worth more
	if deck.Cards[0] >= 1 {
		if value <= 11 { value += 10 }
	}

	return value
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

func (deck Deck) PlayerGameTree(dealer Deck, hand Deck, canDouble bool, splitsLeft int, playerTransposition map[Deck]float64) (float64, string) {
	if hand.Value() > 21 {
		return -1, "bust"
	}

	transpositionEV, contains := playerTransposition[hand]
	if contains {
		return transpositionEV, "?"
	}

	
	// Compute Stand
	standEV := deck.DealerGameTree(dealer, hand.Value(), hand.IsBlackJack(), map[Deck]float64{})

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
		
		expectation, _ := subtreeDeck.PlayerGameTree(dealer, subtreeHand, false, 0, playerTransposition)
		hitEV += probability * expectation
	}

	finalEV := standEV
	finalAction := "stand"
	
	if hitEV > finalEV {
		finalEV = hitEV
		finalAction = "hit"
	}

	if canDouble {
		// Double Down
		doubleEV := 0.0
		for card, amount := range deck.Cards {
			if amount == 0 { continue }
			probability := float64(amount) / size

			subtreeDeck := deck
			subtreeHand := hand

			subtreeDeck.Pull(card)
			subtreeHand.Add(card)
			if subtreeHand.Value() > 21 { 
				doubleEV += probability * -2
			} else {
				expectation := subtreeDeck.DealerGameTree(dealer, subtreeHand.Value(), subtreeHand.IsBlackJack(), map[Deck]float64{})
				doubleEV += probability * 2 * expectation
			}
		}

		if doubleEV > finalEV {
			finalEV = doubleEV
			finalAction = "double"
		}
	}

	if splitsLeft > 0 {
		// Split
		splitEV := -1.0
		for card, amount := range hand.Cards {
			if amount == 2 {
				subtreeDeck := deck
				subtreeHand := hand

				subtreeHand.Pull(card)
				expectation, _ := subtreeDeck.PlayerGameTree(dealer, subtreeHand, doubleAfterSplit, splitsLeft - 1, map[Deck]float64{})
				splitEV = 2 * expectation
				break
			}
		}	

		if splitEV > finalEV {
			finalEV = splitEV
			finalAction = "split"
		}
	}
	
	/*
		fmt.Println("Double", doubleEV)
		fmt.Println("Hit", hitEV)
		fmt.Println("Stand", standEV)
		fmt.Println("Split", splitEV)
		fmt.Println(">", finalAction, finalEV)
		// Surrender is always -0.5 so if EV falls below that
	*/

	playerTransposition[hand] = finalEV
	return finalEV, finalAction
}

func (deck Deck) DealerGameTree(dealer Deck, playerValue int, blackjack bool, dealerTransposition map[Deck]float64) float64 {
	dealerValue := dealer.Value()

	// Dealer goes bust
	if dealerValue > 21 {
		return 1
	}

	if dealerValue == 21 {
		if dealer.IsBlackJack() {
			if blackjack {
				return 0 // If player also gets a blackjack push
			}
			return -1
		}
	
		// If player has a blackjack assume 3:2 payout
		if blackjack {
			return blackjackPayout
		}
	}

	// Dealer stands on 17 and up
	
	if dealerValue >= 17 {
		if dealerValue > playerValue {
			return -1
		} else if dealerValue == playerValue {
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
		
		subtreeEV := subtreeDeck.DealerGameTree(subtreeDealer, playerValue, blackjack, dealerTransposition)
		EV += probability * subtreeEV
	}

	dealerTransposition[dealer] = EV
	return EV
}









func computeHand(playerCard1, playerCard2, dealerCard int, remove bool) (float64, string) {
	testDeck := Shoes(6)

	playerDeck := Deck{}
	dealerDeck := Deck{}

	playerDeck.Add(playerCard1).Add(playerCard2)
	dealerDeck.Add(dealerCard)

	if remove {
		testDeck.Pull(playerCard1).Pull(playerCard2).Pull(dealerCard)
	}

	return testDeck.PlayerGameTree(dealerDeck, playerDeck, true, 2, map[Deck]float64{})
}

func computeBasicStrategy(){
	charmap := map[string]string{
		"split" : "Y",
		"double": "D",
		"hit"   : "H",
		"stand" : "S",
	}

	choiceMatrix := [1000]string{}
	evMatrix := [1000]float64{}

	
	for a:=0;a<=9;a++{
		for b:=0;b<=9;b++{
			for c:=0;c<=9;c++ {
				ev, choice := computeHand(b,c,a, true)
				idx := (100 * a) + (10 * b) + c
				evMatrix[idx] = ev
				choiceMatrix[idx] = charmap[choice] 
			}
		}
	}

	for z:=0;z<=9;z++ {
		fmt.Println("===================")
		fmt.Println("Dealer Upcard: ", string(CONVERT[z]))
		fmt.Println("   A23456789X")
		fmt.Println("")
		for y:=0;y<=9;y++{
			line := string(CONVERT[y]) + "  "
			for x:=0;x<=9;x++{
				idx := (100 * z) + (10 * y) + x
				line += choiceMatrix[idx]
			}
			fmt.Println(line)
		}
	}	


}

func Count(remaining Deck) int {
	key := [10]int{1, -1, -1, -1, -1, -1, 0, 0, 0, 1}
	count := 0

	for i, weight := range key {
		// We multiply by negative weight cause we are using cards in the deck to compute count, not the ones drawn
		count += key[i] * -weight
	} 
	return count
}

func simulation(n int){
	tableDeck := Shoes(6)
	
	for i:=0;i<n;i++ {
		// Reshuffle if less than a deck is left
		if 52 > tableDeck.Size() {
			tableDeck = Shoes(6)
		}
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