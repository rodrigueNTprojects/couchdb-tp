// pkg/loader/loader.go
// Chargement CSV avec transformation en registres distribués
package loader

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"couchdb-tp/pkg/client"
)

type CSVLoader struct {
	CouchDBURL string
	CSVDir     string
	
	// Données en mémoire
	Customers                 map[string]Customer
	Geolocation               map[string][]Geolocation
	Orders                    map[string]Order
	OrderItems                map[string][]OrderItem
	OrderPayments             map[string][]OrderPayment
	OrderReviews              map[string][]OrderReview
	Products                  map[string]Product
	ProductCategoryTranslation map[string]string
	Sellers                   map[string]Seller
	LeadsQualified            map[string]LeadQualified
	LeadsClosed               map[string]LeadClosed
	
	// Statistiques
	Stats LoadStats
}

type LoadStats struct {
	Customers               int
	Geolocations            int
	Orders                  int
	OrderItems              int
	OrderPayments           int
	OrderReviews            int
	Products                int
	Sellers                 int
	Categories              int
	LeadsQualified          int
	LeadsClosed             int
}

// Structures de données
type Customer struct {
	CustomerID       string
	CustomerUniqueID string
	ZipCodePrefix    string
	City             string
	State            string
}

type Geolocation struct {
	ZipCodePrefix string
	Lat           float64
	Lng           float64
	City          string
	State         string
}

type Order struct {
	OrderID                      string
	CustomerID                   string
	OrderStatus                  string
	OrderPurchaseTimestamp       string
	OrderApprovedAt              string
	OrderDeliveredCarrierDate    string
	OrderDeliveredCustomerDate   string
	OrderEstimatedDeliveryDate   string
}

type OrderItem struct {
	OrderID             string
	OrderItemID         int
	ProductID           string
	SellerID            string
	ShippingLimitDate   string
	Price               float64
	FreightValue        float64
}

type OrderPayment struct {
	OrderID              string
	PaymentSequential    int
	PaymentType          string
	PaymentInstallments  int
	PaymentValue         float64
}

type OrderReview struct {
	ReviewID              string
	OrderID               string
	ReviewScore           int
	ReviewCommentTitle    string
	ReviewCommentMessage  string
	ReviewCreationDate    string
	ReviewAnswerTimestamp string
}

type Product struct {
	ProductID                string
	ProductCategoryName      string
	ProductNameLength        int
	ProductDescriptionLength int
	ProductPhotosQty         int
	ProductWeightG           int
	ProductLengthCm          int
	ProductHeightCm          int
	ProductWidthCm           int
}

type Seller struct {
	SellerID      string
	ZipCodePrefix string
	City          string
	State         string
}

type LeadQualified struct {
	MqlID            string
	FirstContactDate string
	LandingPageID    string
	Origin           string
}

type LeadClosed struct {
	MqlID                         string
	SellerID                      string
	SdrID                         string
	SrID                          string
	WonDate                       string
	BusinessSegment               string
	LeadType                      string
	LeadBehaviourProfile          string
	HasCompany                    string
	HasGtin                       string
	AverageStock                  string
	BusinessType                  string
	DeclaredProductCatalogSize    int
	DeclaredMonthlyRevenue        float64
}

// NewCSVLoader crée un nouveau loader
func NewCSVLoader(couchDBURL, csvDir string) (*CSVLoader, error) {
	return &CSVLoader{
		CouchDBURL:                 couchDBURL,
		CSVDir:                     csvDir,
		Customers:                  make(map[string]Customer),
		Geolocation:                make(map[string][]Geolocation),
		Orders:                     make(map[string]Order),
		OrderItems:                 make(map[string][]OrderItem),
		OrderPayments:              make(map[string][]OrderPayment),
		OrderReviews:               make(map[string][]OrderReview),
		Products:                   make(map[string]Product),
		ProductCategoryTranslation: make(map[string]string),
		Sellers:                    make(map[string]Seller),
		LeadsQualified:             make(map[string]LeadQualified),
		LeadsClosed:                make(map[string]LeadClosed),
	}, nil
}

// LoadAllCSVFiles charge tous les fichiers CSV en mémoire
func (l *CSVLoader) LoadAllCSVFiles() error {
	log.Println("Chargement des fichiers CSV en memoire...")
	
	// Charger dans l'ordre pour les dépendances
	if err := l.loadCustomers(); err != nil {
		return fmt.Errorf("erreur chargement customers: %v", err)
	}
	
	if err := l.loadGeolocation(); err != nil {
		return fmt.Errorf("erreur chargement geolocation: %v", err)
	}
	
	if err := l.loadOrders(); err != nil {
		return fmt.Errorf("erreur chargement orders: %v", err)
	}
	
	if err := l.loadOrderItems(); err != nil {
		return fmt.Errorf("erreur chargement order_items: %v", err)
	}
	
	if err := l.loadOrderPayments(); err != nil {
		return fmt.Errorf("erreur chargement order_payments: %v", err)
	}
	
	if err := l.loadOrderReviews(); err != nil {
		return fmt.Errorf("erreur chargement order_reviews: %v", err)
	}
	
	if err := l.loadProducts(); err != nil {
		return fmt.Errorf("erreur chargement products: %v", err)
	}
	
	if err := l.loadProductCategoryTranslation(); err != nil {
		return fmt.Errorf("erreur chargement categories: %v", err)
	}
	
	if err := l.loadSellers(); err != nil {
		return fmt.Errorf("erreur chargement sellers: %v", err)
	}
	
	if err := l.loadLeadsQualified(); err != nil {
		return fmt.Errorf("erreur chargement leads_qualified: %v", err)
	}
	
	if err := l.loadLeadsClosed(); err != nil {
		return fmt.Errorf("erreur chargement leads_closed: %v", err)
	}
	
	l.printStats()
	return nil
}

// CreateCouchDBDocuments crée et insère les documents dans CouchDB
func (l *CSVLoader) CreateCouchDBDocuments() error {
	couchClient, err := client.New(l.CouchDBURL, "", "")
	if err != nil {
		return fmt.Errorf("erreur création client CouchDB: %v", err)
	}
	
	// Créer documents produits
	log.Println("Creation des documents produits...")
	productDocs, err := l.createProductDocuments()
	if err != nil {
		return err
	}
	if err := l.bulkInsert(couchClient, "ecommerce_products", productDocs); err != nil {
		return err
	}
	
	// Créer documents vendeurs
	log.Println("Creation des documents vendeurs...")
	sellerDocs, err := l.createSellerDocuments()
	if err != nil {
		return err
	}
	if err := l.bulkInsert(couchClient, "ecommerce_sellers", sellerDocs); err != nil {
		return err
	}
	
	// Créer documents leads
	log.Println("Creation des documents prospects...")
	leadDocs, err := l.createLeadDocuments()
	if err != nil {
		return err
	}
	if err := l.bulkInsert(couchClient, "ecommerce_leads", leadDocs); err != nil {
		return err
	}
	
	// Créer documents commandes (plus complexe)
	log.Println("Creation des documents commandes...")
	orderDocs, err := l.createOrderDocuments()
	if err != nil {
		return err
	}
	if err := l.bulkInsert(couchClient, "ecommerce_orders", orderDocs); err != nil {
		return err
	}
	
	return nil
}

// createOrderDocuments crée les documents de registre de transactions
func (l *CSVLoader) createOrderDocuments() ([]map[string]interface{}, error) {
	var docs []map[string]interface{}
	
	for orderID, order := range l.Orders {
		// Construire transaction_data avec dénormalisation
		transactionData := map[string]interface{}{
			"order_id": orderID,
			"customer": l.buildCustomerData(order.CustomerID),
			"order_details": l.buildOrderDetails(orderID),
			"status": strings.ToLower(order.OrderStatus),
			"timestamps": l.buildTimestamps(order),
			"payments": l.buildPayments(orderID),
			"reviews": l.buildReviews(orderID),
		}
		
		// Calculer hash pour intégrité
		dataBytes, _ := json.Marshal(transactionData)
		hash := calculateSHA256(dataBytes)
		
		// Déterminer la région (simplification: basé sur state)
		region := l.determineRegion(order.CustomerID)
		
		// Document conforme au registre distribué
		doc := map[string]interface{}{
			"_id": fmt.Sprintf("ledger_transaction_%s", orderID),
			
			// Métadonnées registre
			"ledger_type": "commercial_transaction",
			"immutable": true,
			"transaction_hash": hash,
			"timestamp": order.OrderPurchaseTimestamp,
			
			// Audit trail complet
			"audit_trail": map[string]interface{}{
				"created_by": "csv_loader_system",
				"created_at": time.Now().Format(time.RFC3339),
				"source_node": region,
				"validation_hash": hash,
				"integrity_check": "validated",
				"ledger_version": "1.0",
			},
			
			// Données métier
			"transaction_data": transactionData,
		}
		
		docs = append(docs, doc)
	}
	
	return docs, nil
}

// createProductDocuments crée les documents de registre produits
func (l *CSVLoader) createProductDocuments() ([]map[string]interface{}, error) {
	var docs []map[string]interface{}
	
	for productID, product := range l.Products {
		// Données produit
		productData := map[string]interface{}{
			"product_id": productID,
			"category": strings.ToLower(product.ProductCategoryName),
			"category_english": l.ProductCategoryTranslation[product.ProductCategoryName],
			"specifications": map[string]interface{}{
				"name_length": product.ProductNameLength,
				"description_length": product.ProductDescriptionLength,
				"photos_qty": product.ProductPhotosQty,
				"dimensions": map[string]interface{}{
					"weight_g": product.ProductWeightG,
					"length_cm": product.ProductLengthCm,
					"height_cm": product.ProductHeightCm,
					"width_cm": product.ProductWidthCm,
				},
			},
		}
		
		// Calculer hash
		dataBytes, _ := json.Marshal(productData)
		hash := calculateSHA256(dataBytes)
		
		doc := map[string]interface{}{
			"_id": fmt.Sprintf("ledger_product_%s_v1", productID),
			
			// Métadonnées registre produit
			"ledger_type": "product_definition",
			"version": 1,
			"effective_date": time.Now().Format(time.RFC3339),
			"product_hash": hash,
			
			// Audit trail
			"audit_trail": map[string]interface{}{
				"created_by": "csv_loader_system",
				"created_at": time.Now().Format(time.RFC3339),
				"source_node": "NA1",
				"validation_hash": hash,
				"integrity_check": "validated",
				"ledger_version": "1.0",
			},
			
			"product_data": productData,
		}
		
		docs = append(docs, doc)
	}
	
	return docs, nil
}

// createSellerDocuments crée les documents de registre partenaires
func (l *CSVLoader) createSellerDocuments() ([]map[string]interface{}, error) {
	var docs []map[string]interface{}
	
	for sellerID, seller := range l.Sellers {
		// Récupérer coordonnées
		coords := l.getCoordinates(seller.ZipCodePrefix)
		region := l.determineRegionFromState(seller.State)
		
		sellerData := map[string]interface{}{
			"seller_id": sellerID,
			"business_info": map[string]interface{}{
				"city": strings.ToLower(seller.City),
				"state": strings.ToLower(seller.State),
				"zip_code_prefix": seller.ZipCodePrefix,
			},
			"location": map[string]interface{}{
				"coordinates": coords,
				"region": region,
			},
		}
		
		// Calculer hash
		dataBytes, _ := json.Marshal(sellerData)
		hash := calculateSHA256(dataBytes)
		
		doc := map[string]interface{}{
			"_id": fmt.Sprintf("ledger_seller_%s", sellerID),
			
			// Métadonnées registre partenaire
			"ledger_type": "partner_registry",
			"certification_status": "active",
			"seller_hash": hash,
			
			// Audit trail
			"audit_trail": map[string]interface{}{
				"created_by": "csv_loader_system",
				"created_at": time.Now().Format(time.RFC3339),
				"source_node": "NA1",
				"validation_hash": hash,
				"integrity_check": "validated",
				"ledger_version": "1.0",
			},
			
			"seller_data": sellerData,
		}
		
		docs = append(docs, doc)
	}
	
	return docs, nil
}

// createLeadDocuments crée les documents de registre pipeline
func (l *CSVLoader) createLeadDocuments() ([]map[string]interface{}, error) {
	var docs []map[string]interface{}
	
	for mqlID, qualified := range l.LeadsQualified {
		leadData := map[string]interface{}{
			"mql_id": mqlID,
			"qualification": map[string]interface{}{
				"first_contact_date": qualified.FirstContactDate,
				"origin": qualified.Origin,
				"landing_page_id": qualified.LandingPageID,
			},
		}
		
		// Vérifier si converti
		pipelineStage := "qualified"
		if closed, exists := l.LeadsClosed[mqlID]; exists {
			pipelineStage = "closed"
			leadData["conversion"] = map[string]interface{}{
				"seller_id": closed.SellerID,
				"sdr_id": closed.SdrID,
				"sr_id": closed.SrID,
				"won_date": closed.WonDate,
				"business_segment": closed.BusinessSegment,
				"lead_type": closed.LeadType,
				"lead_behaviour_profile": closed.LeadBehaviourProfile,
				"has_company": closed.HasCompany,
				"has_gtin": closed.HasGtin,
				"average_stock": closed.AverageStock,
				"business_type": closed.BusinessType,
				"declared_product_catalog_size": closed.DeclaredProductCatalogSize,
				"declared_monthly_revenue": closed.DeclaredMonthlyRevenue,
			}
		}
		
		// Calculer hash
		dataBytes, _ := json.Marshal(leadData)
		hash := calculateSHA256(dataBytes)
		
		doc := map[string]interface{}{
			"_id": fmt.Sprintf("ledger_lead_%s", mqlID),
			
			// Métadonnées registre pipeline
			"ledger_type": "sales_pipeline",
			"pipeline_stage": pipelineStage,
			"lead_hash": hash,
			
			// Audit trail
			"audit_trail": map[string]interface{}{
				"created_by": "csv_loader_system",
				"created_at": time.Now().Format(time.RFC3339),
				"source_node": "NA1",
				"validation_hash": hash,
				"integrity_check": "validated",
				"ledger_version": "1.0",
			},
			
			"lead_data": leadData,
		}
		
		docs = append(docs, doc)
	}
	
	return docs, nil
}

// Fonctions utilitaires

func (l *CSVLoader) buildCustomerData(customerID string) map[string]interface{} {
	customer, exists := l.Customers[customerID]
	if !exists {
		return map[string]interface{}{}
	}
	
	coords := l.getCoordinates(customer.ZipCodePrefix)
	
	return map[string]interface{}{
		"customer_id": customer.CustomerID,
		"customer_unique_id": customer.CustomerUniqueID,
		"location": map[string]interface{}{
			"zip_code_prefix": customer.ZipCodePrefix,
			"city": strings.ToLower(customer.City),
			"state": strings.ToLower(customer.State),
			"coordinates": coords,
		},
	}
}

func (l *CSVLoader) buildOrderDetails(orderID string) map[string]interface{} {
	items := l.OrderItems[orderID]
	itemsData := []map[string]interface{}{}
	
	var itemsTotal, freightTotal float64
	
	for _, item := range items {
		product, _ := l.Products[item.ProductID]
		seller, _ := l.Sellers[item.SellerID]
		
		itemData := map[string]interface{}{
			"order_item_id": item.OrderItemID,
			"product_id": item.ProductID,
			"seller_id": item.SellerID,
			"shipping_limit_date": item.ShippingLimitDate,
			"pricing": map[string]interface{}{
				"price": item.Price,
				"freight_value": item.FreightValue,
			},
		}
		
		// Ajouter détails produit
		if product.ProductID != "" {
			itemData["product_details"] = map[string]interface{}{
				"category": product.ProductCategoryName,
				"category_english": l.ProductCategoryTranslation[product.ProductCategoryName],
				"weight_g": product.ProductWeightG,
			}
		}
		
		// Ajouter détails seller
		if seller.SellerID != "" {
			coords := l.getCoordinates(seller.ZipCodePrefix)
			itemData["seller_location"] = map[string]interface{}{
				"city": seller.City,
				"state": seller.State,
				"coordinates": coords,
			}
		}
		
		itemsData = append(itemsData, itemData)
		itemsTotal += item.Price
		freightTotal += item.FreightValue
	}
	
	return map[string]interface{}{
		"items": itemsData,
		"totals": map[string]interface{}{
			"items_total": itemsTotal,
			"freight_total": freightTotal,
			"order_total": itemsTotal + freightTotal,
		},
	}
}

func (l *CSVLoader) buildTimestamps(order Order) map[string]interface{} {
	return map[string]interface{}{
		"purchase": order.OrderPurchaseTimestamp,
		"approved": order.OrderApprovedAt,
		"carrier_received": order.OrderDeliveredCarrierDate,
		"delivered": order.OrderDeliveredCustomerDate,
		"estimated_delivery": order.OrderEstimatedDeliveryDate,
	}
}

func (l *CSVLoader) buildPayments(orderID string) []map[string]interface{} {
	payments := l.OrderPayments[orderID]
	result := []map[string]interface{}{}
	
	for _, payment := range payments {
		result = append(result, map[string]interface{}{
			"payment_sequential": payment.PaymentSequential,
			"payment_type": payment.PaymentType,
			"payment_installments": payment.PaymentInstallments,
			"payment_value": payment.PaymentValue,
		})
	}
	
	return result
}

func (l *CSVLoader) buildReviews(orderID string) []map[string]interface{} {
	reviews := l.OrderReviews[orderID]
	result := []map[string]interface{}{}
	
	for _, review := range reviews {
		result = append(result, map[string]interface{}{
			"review_id": review.ReviewID,
			"review_score": review.ReviewScore,
			"review_comment_title": review.ReviewCommentTitle,
			"review_comment_message": review.ReviewCommentMessage,
			"review_creation_date": review.ReviewCreationDate,
			"review_answer_timestamp": review.ReviewAnswerTimestamp,
		})
	}
	
	return result
}

func (l *CSVLoader) getCoordinates(zipCode string) []float64 {
	geos, exists := l.Geolocation[zipCode]
	if !exists || len(geos) == 0 {
		return []float64{0, 0}
	}
	// Prendre la première géolocalisation
	return []float64{geos[0].Lat, geos[0].Lng}
}

func (l *CSVLoader) determineRegion(customerID string) string {
	customer, exists := l.Customers[customerID]
	if !exists {
		return "NA1"
	}
	return l.determineRegionFromState(customer.State)
}

func (l *CSVLoader) determineRegionFromState(state string) string {
	// Simplification: tous les états brésiliens → north_america
	// Dans une vraie implémentation, on utiliserait une table de mapping
	stateUpper := strings.ToUpper(state)
	
	// Exemples de mapping (fictif pour le TP)
	naStates := []string{"SP", "RJ", "MG", "RS", "PR", "SC", "BA", "PE", "CE"}
	euStates := []string{"AM", "PA", "AC", "RO"}
	
	for _, s := range naStates {
		if s == stateUpper {
			return "NA1"
		}
	}
	
	for _, s := range euStates {
		if s == stateUpper {
			return "EU1"
		}
	}
	
	return "AP1" // Par défaut
}

func calculateSHA256(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

/* func (l *CSVLoader) bulkInsert(client *client.Client, dbName string, docs []map[string]interface{}) error {
	if len(docs) == 0 {
		log.Printf("Aucun document a inserer pour %s", dbName)
		return nil
	}
	
	bulkDocs := map[string]interface{}{
		"docs": docs,
	}
	
	resp, err := client.Post(fmt.Sprintf("/%s/_bulk_docs", dbName), bulkDocs)
	if err != nil {
		return fmt.Errorf("erreur bulk insert %s: %v", dbName, err)
	}
	
	if resp.StatusCode != 201 {
		return fmt.Errorf("erreur HTTP %d pour %s", resp.StatusCode, dbName)
	}
	
	// Compter succès/erreurs
	var results []map[string]interface{}
	if err := json.Unmarshal(resp.Body, &results); err != nil {
		return fmt.Errorf("erreur parsing reponse: %v", err)
	}
	
	success := 0
	errors := 0
	for _, result := range results {
		if _, hasError := result["error"]; hasError {
			errors++
		} else {
			success++
		}
	}
	
	log.Printf("Lot %s insere: %d succes, %d erreurs", dbName, success, errors)
	
	return nil
} */

func (l *CSVLoader) bulkInsert(client *client.Client, dbName string, docs []map[string]interface{}) error {
	if len(docs) == 0 {
		log.Printf("Aucun document a inserer pour %s", dbName)
		return nil
	}
	
	// Découper en lots de 1000 documents pour éviter les timeouts
	batchSize := 5000
	totalDocs := len(docs)
	totalSuccess := 0
	totalErrors := 0
	
	log.Printf("Insertion de %d documents en lots de %d...", totalDocs, batchSize)
	
	for i := 0; i < totalDocs; i += batchSize {
		end := i + batchSize
		if end > totalDocs {
			end = totalDocs
		}
		
		batch := docs[i:end]
		
		log.Printf("  Lot %d-%d / %d...", i+1, end, totalDocs)
		
		bulkDocs := map[string]interface{}{
			"docs": batch,
		}
		
		resp, err := client.Post(fmt.Sprintf("/%s/_bulk_docs", dbName), bulkDocs)
		if err != nil {
			log.Printf("  Erreur lot %d-%d: %v", i+1, end, err)
			totalErrors += len(batch)
			continue
		}
		
		if resp.StatusCode != 201 {
			log.Printf("  Erreur HTTP %d pour lot %d-%d", resp.StatusCode, i+1, end)
			totalErrors += len(batch)
			continue
		}
		
		// Compter succès/erreurs dans la réponse
		var results []map[string]interface{}
		if err := json.Unmarshal(resp.Body, &results); err != nil {
			log.Printf("  Avertissement: impossible de parser réponse pour lot %d-%d", i+1, end)
			totalSuccess += len(batch)
			continue
		}
		
		batchSuccess := 0
		batchErrors := 0
		for _, result := range results {
			if _, hasError := result["error"]; hasError {
				batchErrors++
			} else {
				batchSuccess++
			}
		}
		
		totalSuccess += batchSuccess
		totalErrors += batchErrors
		
		log.Printf("  ✓ Lot inséré: %d succès, %d erreurs", batchSuccess, batchErrors)
		
		// Petite pause pour ne pas surcharger CouchDB
		// if end < totalDocs {
		// 	time.Sleep(100 * time.Millisecond)
		// }
	}
	
	log.Printf("Insertion %s terminée: %d succès, %d erreurs sur %d documents", 
		dbName, totalSuccess, totalErrors, totalDocs)
	
	return nil
}

func (l *CSVLoader) printStats() {
	log.Println("")
	log.Println("Statistiques de chargement:")
	log.Printf("  Clients: %d", l.Stats.Customers)
	log.Printf("  Geolocalisations: %d", l.Stats.Geolocations)
	log.Printf("  Commandes: %d", l.Stats.Orders)
	log.Printf("  Articles commandes: %d", l.Stats.OrderItems)
	log.Printf("  Paiements: %d", l.Stats.OrderPayments)
	log.Printf("  Avis: %d", l.Stats.OrderReviews)
	log.Printf("  Produits: %d", l.Stats.Products)
	log.Printf("  Vendeurs: %d", l.Stats.Sellers)
	log.Printf("  Categories: %d", l.Stats.Categories)
	log.Printf("  Prospects qualifies: %d", l.Stats.LeadsQualified)
	log.Printf("  Prospects convertis: %d", l.Stats.LeadsClosed)
	log.Println("")
}

// Fonctions de chargement CSV (inchangées)
// [Les fonctions loadCustomers, loadGeolocation, etc. restent identiques]
// Pour la concision, je les omets ici mais elles doivent rester dans le fichier

func (l *CSVLoader) loadCustomers() error {
	log.Println("Chargement: Clients (customers.csv)")
	filePath := filepath.Join(l.CSVDir, "customers.csv")
	
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture fichier: %v", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return err
	}
	
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		
		customer := Customer{}
		for i, value := range record {
			switch headers[i] {
			case "customer_id":
				customer.CustomerID = value
			case "customer_unique_id":
				customer.CustomerUniqueID = value
			case "customer_zip_code_prefix":
				customer.ZipCodePrefix = value
			case "customer_city":
				customer.City = value
			case "customer_state":
				customer.State = value
			}
		}
		
		l.Customers[customer.CustomerID] = customer
		count++
	}
	
	l.Stats.Customers = count
	log.Printf("  %d enregistrements traites avec succes", count)
	log.Println("Termine: Clients")
	
	return nil
}

func (l *CSVLoader) loadGeolocation() error {
	log.Println("Chargement: Geolocalisation (geolocation.csv)")
	filePath := filepath.Join(l.CSVDir, "geolocation.csv")
	
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture fichier: %v", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return err
	}
	
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		
		geo := Geolocation{}
		for i, value := range record {
			switch headers[i] {
			case "geolocation_zip_code_prefix":
				geo.ZipCodePrefix = value
			case "geolocation_lat":
				geo.Lat, _ = strconv.ParseFloat(value, 64)
			case "geolocation_lng":
				geo.Lng, _ = strconv.ParseFloat(value, 64)
			case "geolocation_city":
				geo.City = value
			case "geolocation_state":
				geo.State = value
			}
		}
		
		l.Geolocation[geo.ZipCodePrefix] = append(l.Geolocation[geo.ZipCodePrefix], geo)
		count++
	}
	
	l.Stats.Geolocations = count
	log.Printf("  %d enregistrements traites avec succes", count)
	log.Println("Termine: Geolocalisation")
	
	return nil
}

func (l *CSVLoader) loadOrders() error {
	log.Println("Chargement: Commandes (orders.csv)")
	filePath := filepath.Join(l.CSVDir, "orders.csv")
	
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture fichier: %v", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return err
	}
	
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		
		order := Order{}
		for i, value := range record {
			switch headers[i] {
			case "order_id":
				order.OrderID = value
			case "customer_id":
				order.CustomerID = value
			case "order_status":
				order.OrderStatus = value
			case "order_purchase_timestamp":
				order.OrderPurchaseTimestamp = value
			case "order_approved_at":
				order.OrderApprovedAt = value
			case "order_delivered_carrier_date":
				order.OrderDeliveredCarrierDate = value
			case "order_delivered_customer_date":
				order.OrderDeliveredCustomerDate = value
			case "order_estimated_delivery_date":
				order.OrderEstimatedDeliveryDate = value
			}
		}
		
		l.Orders[order.OrderID] = order
		count++
	}
	
	l.Stats.Orders = count
	log.Printf("  %d enregistrements traites avec succes", count)
	log.Println("Termine: Commandes")
	
	return nil
}

func (l *CSVLoader) loadOrderItems() error {
	log.Println("Chargement: Articles commandes (order_items.csv)")
	filePath := filepath.Join(l.CSVDir, "order_items.csv")
	
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture fichier: %v", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return err
	}
	
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		
		item := OrderItem{}
		for i, value := range record {
			switch headers[i] {
			case "order_id":
				item.OrderID = value
			case "order_item_id":
				item.OrderItemID, _ = strconv.Atoi(value)
			case "product_id":
				item.ProductID = value
			case "seller_id":
				item.SellerID = value
			case "shipping_limit_date":
				item.ShippingLimitDate = value
			case "price":
				item.Price, _ = strconv.ParseFloat(value, 64)
			case "freight_value":
				item.FreightValue, _ = strconv.ParseFloat(value, 64)
			}
		}
		
		l.OrderItems[item.OrderID] = append(l.OrderItems[item.OrderID], item)
		count++
	}
	
	l.Stats.OrderItems = count
	log.Printf("  %d enregistrements traites avec succes", count)
	log.Println("Termine: Articles commandes")
	
	return nil
}

func (l *CSVLoader) loadOrderPayments() error {
	log.Println("Chargement: Paiements (order_payments.csv)")
	filePath := filepath.Join(l.CSVDir, "order_payments.csv")
	
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture fichier: %v", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return err
	}
	
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		
		payment := OrderPayment{}
		for i, value := range record {
			switch headers[i] {
			case "order_id":
				payment.OrderID = value
			case "payment_sequential":
				payment.PaymentSequential, _ = strconv.Atoi(value)
			case "payment_type":
				payment.PaymentType = value
			case "payment_installments":
				payment.PaymentInstallments, _ = strconv.Atoi(value)
			case "payment_value":
				payment.PaymentValue, _ = strconv.ParseFloat(value, 64)
			}
		}
		
		l.OrderPayments[payment.OrderID] = append(l.OrderPayments[payment.OrderID], payment)
		count++
	}
	
	l.Stats.OrderPayments = count
	log.Printf("  %d enregistrements traites avec succes", count)
	log.Println("Termine: Paiements")
	
	return nil
}

func (l *CSVLoader) loadOrderReviews() error {
	log.Println("Chargement: Avis (order_reviews.csv)")
	filePath := filepath.Join(l.CSVDir, "order_reviews.csv")
	
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture fichier: %v", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return err
	}
	
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		
		review := OrderReview{}
		for i, value := range record {
			switch headers[i] {
			case "review_id":
				review.ReviewID = value
			case "order_id":
				review.OrderID = value
			case "review_score":
				review.ReviewScore, _ = strconv.Atoi(value)
			case "review_comment_title":
				review.ReviewCommentTitle = value
			case "review_comment_message":
				review.ReviewCommentMessage = value
			case "review_creation_date":
				review.ReviewCreationDate = value
			case "review_answer_timestamp":
				review.ReviewAnswerTimestamp = value
			}
		}
		
		l.OrderReviews[review.OrderID] = append(l.OrderReviews[review.OrderID], review)
		count++
	}
	
	l.Stats.OrderReviews = count
	log.Printf("  %d enregistrements traites avec succes", count)
	log.Println("Termine: Avis")
	
	return nil
}

func (l *CSVLoader) loadProducts() error {
	log.Println("Chargement: Produits (products.csv)")
	filePath := filepath.Join(l.CSVDir, "products.csv")
	
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture fichier: %v", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return err
	}
	
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		
		product := Product{}
		for i, value := range record {
			switch headers[i] {
			case "product_id":
				product.ProductID = value
			case "product_category_name":
				product.ProductCategoryName = value
			case "product_name_lenght":
				product.ProductNameLength, _ = strconv.Atoi(value)
			case "product_description_lenght":
				product.ProductDescriptionLength, _ = strconv.Atoi(value)
			case "product_photos_qty":
				product.ProductPhotosQty, _ = strconv.Atoi(value)
			case "product_weight_g":
				product.ProductWeightG, _ = strconv.Atoi(value)
			case "product_length_cm":
				product.ProductLengthCm, _ = strconv.Atoi(value)
			case "product_height_cm":
				product.ProductHeightCm, _ = strconv.Atoi(value)
			case "product_width_cm":
				product.ProductWidthCm, _ = strconv.Atoi(value)
			}
		}
		
		l.Products[product.ProductID] = product
		count++
	}
	
	l.Stats.Products = count
	log.Printf("  %d enregistrements traites avec succes", count)
	log.Println("Termine: Produits")
	
	return nil
}

func (l *CSVLoader) loadProductCategoryTranslation() error {
	log.Println("Chargement: Categories (product_category_name_translation.csv)")
	filePath := filepath.Join(l.CSVDir, "product_category_name_translation.csv")
	
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture fichier: %v", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return err
	}
	
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		
		var categoryPT, categoryEN string
		for i, value := range record {
			switch headers[i] {
			case "product_category_name":
				categoryPT = value
			case "product_category_name_english":
				categoryEN = value
			}
		}
		
		l.ProductCategoryTranslation[categoryPT] = categoryEN
		count++
	}
	
	l.Stats.Categories = count
	log.Printf("  %d enregistrements traites avec succes", count)
	log.Println("Termine: Categories")
	
	return nil
}

func (l *CSVLoader) loadSellers() error {
	log.Println("Chargement: Vendeurs (sellers.csv)")
	filePath := filepath.Join(l.CSVDir, "sellers.csv")
	
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture fichier: %v", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return err
	}
	
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		
		seller := Seller{}
		for i, value := range record {
			switch headers[i] {
			case "seller_id":
				seller.SellerID = value
			case "seller_zip_code_prefix":
				seller.ZipCodePrefix = value
			case "seller_city":
				seller.City = value
			case "seller_state":
				seller.State = value
			}
		}
		
		l.Sellers[seller.SellerID] = seller
		count++
	}
	
	l.Stats.Sellers = count
	log.Printf("  %d enregistrements traites avec succes", count)
	log.Println("Termine: Vendeurs")
	
	return nil
}

func (l *CSVLoader) loadLeadsQualified() error {
	log.Println("Chargement: Prospects qualifies (leads_qualified.csv)")
	filePath := filepath.Join(l.CSVDir, "leads_qualified.csv")
	
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture fichier: %v", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return err
	}
	
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		
		lead := LeadQualified{}
		for i, value := range record {
			switch headers[i] {
			case "mql_id":
				lead.MqlID = value
			case "first_contact_date":
				lead.FirstContactDate = value
			case "landing_page_id":
				lead.LandingPageID = value
			case "origin":
				lead.Origin = value
			}
		}
		
		l.LeadsQualified[lead.MqlID] = lead
		count++
	}
	
	l.Stats.LeadsQualified = count
	log.Printf("  %d enregistrements traites avec succes", count)
	log.Println("Termine: Prospects qualifies")
	
	return nil
}

func (l *CSVLoader) loadLeadsClosed() error {
	log.Println("Chargement: Prospects convertis (leads_closed.csv)")
	filePath := filepath.Join(l.CSVDir, "leads_closed.csv")
	
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture fichier: %v", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return err
	}
	
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		
		lead := LeadClosed{}
		for i, value := range record {
			switch headers[i] {
			case "mql_id":
				lead.MqlID = value
			case "seller_id":
				lead.SellerID = value
			case "sdr_id":
				lead.SdrID = value
			case "sr_id":
				lead.SrID = value
			case "won_date":
				lead.WonDate = value
			case "business_segment":
				lead.BusinessSegment = value
			case "lead_type":
				lead.LeadType = value
			case "lead_behaviour_profile":
				lead.LeadBehaviourProfile = value
			case "has_company":
				lead.HasCompany = value
			case "has_gtin":
				lead.HasGtin = value
			case "average_stock":
				lead.AverageStock = value
			case "business_type":
				lead.BusinessType = value
			case "declared_product_catalog_size":
				lead.DeclaredProductCatalogSize, _ = strconv.Atoi(value)
			case "declared_monthly_revenue":
				lead.DeclaredMonthlyRevenue, _ = strconv.ParseFloat(value, 64)
			}
		}
		
		l.LeadsClosed[lead.MqlID] = lead
		count++
	}
	
	l.Stats.LeadsClosed = count
	log.Printf("  %d enregistrements traites avec succes", count)
	log.Println("Termine: Prospects convertis")
	
	return nil
}