package main

//Andrew McCauley

import (
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Set IP Address/Port
var IP_ADDRESS = "localhost:8080"

// Set up receipt and point array "storage"

//receipt data not necessarily required, as only points are retrieved
// var receipts = make(map[string]Receipt)

var pointsArr = make(map[string]Points)

// Receipt and Points Structures
type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
}
type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type Points struct {
	Points int `json:"points"`
}

// Calculates points earned
func getPoints(r Receipt) int {

	pointSum := 0

	// One point for every char in retailer name
	for _, char := range r.Retailer {
		if unicode.IsLetter(char) || unicode.IsNumber(char) {
			pointSum += 1
		}
	}

	if total, err := strconv.ParseFloat(r.Total, 64); err == nil {
		if math.Mod(total, 1) == 0 { //50 points if round dollar amount
			pointSum += 50
		}
		if math.Mod(total, 0.25) == 0 { //25 points if total is multiple of 0.25
			pointSum += 25
		}
	}

	// 5 points for every two items
	pointSum += 5 * int(math.Floor(float64(len(r.Items))/2))

	// Points from each item description multiple of 3
	for _, item := range r.Items {
		if len(strings.TrimSpace(item.ShortDescription))%3 == 0 {
			if price, err := strconv.ParseFloat(item.Price, 64); err == nil {
				pointSum += int(math.Ceil(price * 0.2))
			}

		}
	}

	//6 points if day is odd
	if day, err := strconv.Atoi(strings.Split(r.PurchaseDate, "-")[2]); err == nil && day%2 == 1 {
		pointSum += 6
	}

	timeSplit := strings.Split(r.PurchaseTime, ":")

	//10 points if between 2pm and 4pm (14:01-15:59)
	hr, err1 := strconv.Atoi(timeSplit[0])
	mn, err2 := strconv.Atoi(timeSplit[1])
	//		 		accomodates for 2:01-2:59					then for 3:00-3:59
	if (err1 == nil && err2 == nil) && ((hr == 14 && mn > 0) || (hr == 15)) {
		pointSum += 10
	}

	return pointSum
}

// Adds receipt
func addReceipt(c *gin.Context) {
	var newReceipt Receipt

	if err := c.BindJSON(&newReceipt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"description": "The receipt is invalid."})
		return
	}
	//Validation, ensuring valid data input
	//Retailer
	if m, _ := regexp.MatchString("^[\\w\\s\\-&]+$", newReceipt.Retailer); !m {
		c.JSON(http.StatusBadRequest, gin.H{"description": "The receipt is invalid."})
		return
	}

	//Purchase Date
	dateExample := "2006-01-02"
	if _, err := time.Parse(dateExample, newReceipt.PurchaseDate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"description": "The receipt is invalid."})
		return
	}

	//Purchase Time
	timeExample := "15:04"
	if _, err := time.Parse(timeExample, newReceipt.PurchaseTime); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"description": "The receipt is invalid."})
		return
	}

	//Items
	if len(newReceipt.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"description": "The receipt is invalid."})
		return
	}

	for _, item := range newReceipt.Items {
		descriptionStr, _ := regexp.Compile(`^[\w\s\-]+$`)
		if !descriptionStr.MatchString(item.ShortDescription) {
			c.JSON(http.StatusBadRequest, gin.H{"description": "The receipt is invalid."})
			return
		}
		priceStr, _ := regexp.Compile(`^\d+\.\d{2}$`)
		if !priceStr.MatchString(item.Price) {
			c.JSON(http.StatusBadRequest, gin.H{"description": "The receipt is invalid."})
			return
		}
	}

	//Total
	if m, _ := regexp.MatchString("^\\d+\\.\\d{2}$", newReceipt.Total); !m {
		c.JSON(http.StatusBadRequest, gin.H{"description": "The receipt is invalid."})
		return
	}

	receiptID := uuid.NewString()

	//Used if storing receipt is needed, which it is not in this case as only the points value is required
	//receipts[receiptID] = newReceipt

	// Get points now and save in memory to be called later
	var newPoints Points
	newPoints.Points = getPoints(newReceipt)
	pointsArr[receiptID] = newPoints

	c.JSON(http.StatusOK, gin.H{"id": receiptID})
}

// Gets points for receipt
func receiptPoints(c *gin.Context) {
	id := c.Param("id")
	if points, ok := pointsArr[id]; ok {
		c.JSON(http.StatusOK, points)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"description": "No receipt found for that ID."})
	}

}

// Start Web-Service and Set Routes
func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.POST("/receipts/process", addReceipt)
	router.GET("/receipts/:id/points", receiptPoints)
	println("Receipt API running on " + IP_ADDRESS)
	router.Run(IP_ADDRESS)
}
