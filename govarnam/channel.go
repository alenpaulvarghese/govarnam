package govarnam

import (
	"context"
	"fmt"
)

type channelDictionaryResult struct {
	exactMatches []Suggestion
	suggestions  []Suggestion
}

func (varnam *Varnam) channelTokenizeWord(ctx context.Context, word string, matchType int, partial bool, channel chan *[]Token) {
	select {
	case <-ctx.Done():
		close(channel)
		return
	default:
		channel <- varnam.tokenizeWord(ctx, word, matchType, partial)
		close(channel)
	}
}

func (varnam *Varnam) channelTokensToSuggestions(ctx context.Context, tokens *[]Token, limit int, channel chan []Suggestion) {
	select {
	case <-ctx.Done():
		close(channel)
		return
	default:
		channel <- varnam.tokensToSuggestions(ctx, tokens, false, limit)
		close(channel)
	}
}

func (varnam *Varnam) channelTokensToGreedySuggestions(ctx context.Context, tokens *[]Token, channel chan []Suggestion) {
	select {
	case <-ctx.Done():
		close(channel)
		return
	default:
		// Altering tokens directly will affect others
		tokensCopy := make([]Token, len(*tokens))
		copy(tokensCopy, *tokens)

		tokensCopy = removeNonExactTokens(tokensCopy)

		if len(tokensCopy) == 0 {
			var result []Suggestion
			channel <- result
			close(channel)
			return
		}

		channel <- varnam.tokensToSuggestions(ctx, &tokensCopy, false, varnam.TokenizerSuggestionsLimit)
		tokensCopy = nil
		close(channel)
	}
}

func (varnam *Varnam) channelGetFromDictionary(ctx context.Context, word string, tokens *[]Token, channel chan channelDictionaryResult) {
	var (
		dictResults  []Suggestion
		exactMatches []Suggestion
	)

	select {
	case <-ctx.Done():
		close(channel)
		return
	default:
		dictSugs := varnam.getFromDictionary(ctx, tokens)

		if varnam.Debug {
			fmt.Println("Dictionary results:", dictSugs)
		}

		if len(dictSugs.sugs) > 0 {
			if dictSugs.exactMatch == false {
				// These will be partial words
				restOfWord := word[dictSugs.longestMatchPosition+1:]
				dictResults = varnam.tokenizeRestOfWord(ctx, restOfWord, dictSugs.sugs, varnam.DictionarySuggestionsLimit)
			} else {
				exactMatches = dictSugs.sugs

				// Since partial words are in dictionary, exactMatch will be TRUE
				// for pathway to a word. Hence we're calling this here
				moreFromDict := varnam.getMoreFromDictionary(ctx, dictSugs.sugs)

				if varnam.Debug {
					fmt.Println("More dictionary results:", moreFromDict)
				}

				for _, sugSet := range moreFromDict {
					dictResults = append(dictResults, sugSet...)
				}
			}
		}

		channel <- channelDictionaryResult{exactMatches, dictResults}
		close(channel)
	}
}

func (varnam *Varnam) channelGetFromPatternDictionary(ctx context.Context, word string, channel chan channelDictionaryResult) {
	var (
		dictResults  []Suggestion
		exactMatches []Suggestion
	)

	select {
	case <-ctx.Done():
		close(channel)
		return
	default:
		patternDictSugs := varnam.getFromPatternDictionary(ctx, word)

		if len(patternDictSugs) > 0 {
			if varnam.Debug {
				fmt.Println("Pattern dictionary results:", patternDictSugs)
			}

			var partialMatches []PatternDictionarySuggestion

			for _, match := range patternDictSugs {
				if match.Length < len(word) {
					sug := &match.Sug

					// Increase weight on length matched.
					// 50 because half of 100%
					sug.Weight += match.Length * 50

					for _, cb := range varnam.PatternWordPartializers {
						cb(sug)
					}

					partialMatches = append(partialMatches, match)
				} else if match.Length == len(word) {
					// Same length
					exactMatches = append(exactMatches, match.Sug)
				} else {
					dictResults = append(dictResults, match.Sug)
				}
			}

			perMatchLimit := varnam.PatternDictionarySuggestionsLimit

			if len(partialMatches) > 0 && perMatchLimit > len(partialMatches) {
				perMatchLimit = perMatchLimit / len(partialMatches)
			}

			for _, match := range partialMatches {
				restOfWord := word[match.Length:]

				filled := varnam.tokenizeRestOfWord(ctx, restOfWord, []Suggestion{match.Sug}, perMatchLimit)

				dictResults = append(dictResults, filled...)

				if len(dictResults) >= varnam.PatternDictionarySuggestionsLimit {
					break
				}
			}
		}

		channel <- channelDictionaryResult{exactMatches, dictResults}
		close(channel)
	}
}

func (varnam *Varnam) channelGetMoreFromDictionary(ctx context.Context, sugs []Suggestion, channel chan [][]Suggestion) {
	select {
	case <-ctx.Done():
		close(channel)
		return
	default:
		channel <- varnam.getMoreFromDictionary(ctx, sugs)
		close(channel)
	}
}
