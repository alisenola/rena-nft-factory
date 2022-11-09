package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	collectionDB "rena-nft-factory/collection"
	"rena-nft-factory/database"
	itemDB "rena-nft-factory/item"
	"rena-nft-factory/model"

	"github.com/gosimple/slug"
)

type TokenResponse struct {
	Status string `json:"status"`
	Data   struct {
		Tokens []Item `json:"tokens"`
		Total  int64  `json:"total"`
	}
}

type MetaData struct {
	Compiler     string `json:"compiler"`
	Description  string `json:"description"`
	DNA          string `json:"dna"`
	Image        string `json:"image"`
	Date         string `json:"date"`
	Royalty      int    `json:"royalty"`
	FeeRecipient string `json:"fee_recipient"`
	Category     string `json:"category"`
	Website      string `json:"website"`
	Discord      string `json:"discord"`
	TwitterUrl   string `json:"twitter_url"`
	InstagramUrl string `json:"instagram_url"`
	MediumUrl    string `json:"medium_url"`
	TelegramUrl  string `json:"telegram_url"`
	ContactEmail string `json:"contact_email"`
}

type ItemResponse struct {
	Item Item `json:"item"`
}

type ItemsResponse struct {
	Items      []Item `json:"items"`
	ItemsCount int64  `json:"itemsCount"`
}

type Item struct {
	Id                        uint    `json:"id"`
	ContentType               string  `json:"contentType"`
	ContractAddress           string  `json:"contractAddress"`
	ImageURL                  string  `json:"imageUrl"`
	IsAppropriate             string  `json:"isAppropriate"`
	LastSalePrice             float32 `json:"lastSalePrice"`
	LastSalePriceInUSD        float32 `json:"lastSalePriceInUSD"`
	LastSalePricePaymentToken string  `json:"lastSalePricePaymentToken"`
	Liked                     int     `json:"liked"`
	Name                      string  `json:"name"`
	PaymentToken              string  `json:"paymentToken"`
	Price                     float32 `json:"price"`
	PriceInUSD                float32 `json:"priceInUSD"`
	Supply                    int     `json:"supply"`
	ThumbnailPath             string  `json:"thumbnailPath"`
	TokenID                   int     `json:"tokenId"`
	TokenType                 int     `json:"tokenType"`
	TokenURI                  string  `json:"tokenURI"`
}

func request(target string, from int, count int, sortBy string, cc chan string) {
	query := fmt.Sprintf(`{"from":"%d", "count":"%d", "type":"%s", "sortby":"%s"}`, from, count, target, sortBy)
	var jsonStr = []byte(query)
	req, err := http.NewRequest("POST", "https://fetch-tokens.vercel.app/api", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	cc <- string(body)
}

func main() {
	db, _ := database.NewDatabase()
	collectionDB := collectionDB.NewCollectionDB(db)
	itemDB := itemDB.NewItemDB(db)

	cc := make(chan string, 1)
	go request("all", 0, 1, "createdAt", cc)
	msg := <-cc
	var token TokenResponse
	json.Unmarshal([]byte(msg), &token)

	fmt.Println(token.Data.Total)
	var meta MetaData
	re := regexp.MustCompile(`^(http:\/\/www\.|https:\/\/www\.|http:\/\/|https:\/\/)?[a-z0-9]+([\-\.]{1}[a-z0-9]+)*\.[a-z]{2,5}(:[0-9]{1,5})?(\/.*)?$`)

	for index := 1; index < int(token.Data.Total/6); index++ {
		go func() {
			go request("all", 6*index, 6, "createdAt", cc)
			msg = <-cc
			json.Unmarshal([]byte(msg), &token)

			for _, item := range token.Data.Tokens {
				token := itemDB.FindItemByAddress(item.ContractAddress, item.TokenID)

				fmt.Println(item.Name)
				if token == nil {
					var collection model.Collection
					// Save collection
					tokenURL := item.TokenURI
					if !re.MatchString(item.TokenURI) {
						tokenURL = strings.Replace(tokenURL, "ipfs://", "https://ipfs.io/ipfs/", -1)
					}
					if tokenURL == "ipfs://" {
						continue
					}

					resp, err := http.Get(tokenURL)
					if err != nil {
						fmt.Println(err)
						continue
					}
					defer resp.Body.Close()
					body, err := io.ReadAll(resp.Body)

					json.Unmarshal([]byte(body), &meta)
					if err != nil {
						continue
					}

					if meta.Compiler != "" {
						col := collectionDB.FindCollectionBySlug(slug.Make(meta.Compiler))

						if col == nil {
							fmt.Println(meta.Compiler)
							// save collection
							collection = model.Collection{
								ContractAddress: meta.DNA,
								Slug:            slug.Make(meta.Compiler),
								Name:            meta.Compiler,
								Description:     meta.Description,
								ImageUrl:        meta.Image,
								Royalty:         meta.Royalty,
								FeeRecipient:    meta.FeeRecipient,
								Category:        meta.Category,
								Website:         meta.Website,
								Discord:         meta.Discord,
								TwitterUrl:      meta.TwitterUrl,
								InstagramUrl:    meta.InstagramUrl,
								MediumUrl:       meta.MediumUrl,
								TelegramUrl:     meta.TelegramUrl,
								ContactEmail:    meta.ContactEmail,
							}

							err = collectionDB.SaveCollection(&collection)
							if err != nil {
								if database.IsKeyConflictErr(err) {
									fmt.Println("Duplicated collection name")
								}
								continue
							}
						} else {
							collection = *col
						}
					}

					// save item
					nft := model.Item{
						ContentType:               item.ContentType,
						ContractAddress:           item.ContractAddress,
						ImageURL:                  item.ImageURL,
						IsAppropriate:             item.IsAppropriate,
						LastSalePrice:             item.LastSalePrice,
						LastSalePriceInUSD:        item.LastSalePriceInUSD,
						LastSalePricePaymentToken: item.LastSalePricePaymentToken,
						Liked:                     item.Liked,
						Name:                      item.Name,
						PaymentToken:              item.PaymentToken,
						Price:                     item.Price,
						PriceInUSD:                item.PriceInUSD,
						Supply:                    item.Supply,
						ThumbnailPath:             item.ThumbnailPath,
						TokenID:                   item.TokenID,
						TokenType:                 item.TokenType,
						TokenURI:                  item.TokenURI,
						CollectionId:              collection.Id,
					}

					err = itemDB.SaveItem(&nft)
					if err != nil {
						if database.IsKeyConflictErr(err) {
							fmt.Println("Duplicated collection name")
						}
						continue
					}
				}
			}
		}()
		time.Sleep(3 * time.Second)
	}
}
