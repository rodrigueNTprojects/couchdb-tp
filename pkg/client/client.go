// pkg/client/client.go
// Client HTTP pour communiquer avec CouchDB
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	Username   string
	Password   string
	HTTPClient *http.Client
}

type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// New crée un nouveau client CouchDB
func New(baseURL, username, password string) (*Client, error) {
	// Nettoyer l'URL
	baseURL = strings.TrimSuffix(baseURL, "/")
	
	// Si l'URL contient déjà les credentials, les extraire
	if strings.Contains(baseURL, "@") {
		parts := strings.Split(baseURL, "@")
		if len(parts) == 2 {
			authParts := strings.Split(strings.Replace(parts[0], "http://", "", 1), ":")
			if len(authParts) == 2 {
				username = authParts[0]
				password = authParts[1]
				baseURL = "http://" + parts[1]
			}
		}
	}
	
	return &Client{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second, // Augmenté de 30s à 120s pour les gros lots
		},
	}, nil
}

// Get effectue une requête GET
func (c *Client) Get(path string) (*Response, error) {
	url := c.buildURL(path)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erreur création requête: %v", err)
	}
	
	c.addAuth(req)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur requête: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture réponse: %v", err)
	}
	
	return &Response{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
	}, nil
}

// Post effectue une requête POST
func (c *Client) Post(path string, data interface{}) (*Response, error) {
	url := c.buildURL(path)
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("erreur sérialisation JSON: %v", err)
	}
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("erreur création requête: %v", err)
	}
	
	c.addAuth(req)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur requête: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture réponse: %v", err)
	}
	
	return &Response{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
	}, nil
}

// Put effectue une requête PUT
func (c *Client) Put(path string, data interface{}) (*Response, error) {
	url := c.buildURL(path)
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("erreur sérialisation JSON: %v", err)
	}
	
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("erreur création requête: %v", err)
	}
	
	c.addAuth(req)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur requête: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture réponse: %v", err)
	}
	
	return &Response{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
	}, nil
}

// Delete effectue une requête DELETE
func (c *Client) Delete(path string) (*Response, error) {
	url := c.buildURL(path)
	
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erreur création requête: %v", err)
	}
	
	c.addAuth(req)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur requête: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture réponse: %v", err)
	}
	
	return &Response{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
	}, nil
}

// buildURL construit l'URL complète
func (c *Client) buildURL(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return c.BaseURL + path
}

// addAuth ajoute l'authentification à la requête
func (c *Client) addAuth(req *http.Request) {
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}
}

// TestConnection teste la connexion au serveur CouchDB
func (c *Client) TestConnection() error {
	resp, err := c.Get("/")
	if err != nil {
		return fmt.Errorf("impossible de se connecter à CouchDB: %v", err)
	}
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("réponse inattendue de CouchDB: %d", resp.StatusCode)
	}
	
	return nil
}

// DatabaseExists vérifie si une base de données existe
func (c *Client) DatabaseExists(dbName string) (bool, error) {
	resp, err := c.Get("/" + dbName)
	if err != nil {
		return false, err
	}
	
	return resp.StatusCode == 200, nil
}

// CreateDatabase crée une base de données
func (c *Client) CreateDatabase(dbName string) error {
	resp, err := c.Put("/"+dbName, nil)
	if err != nil {
		return err
	}
	
	if resp.StatusCode != 201 && resp.StatusCode != 412 {
		return fmt.Errorf("erreur création base de données %s: status %d", dbName, resp.StatusCode)
	}
	
	return nil
}

// DeleteDatabase supprime une base de données
func (c *Client) DeleteDatabase(dbName string) error {
	resp, err := c.Delete("/" + dbName)
	if err != nil {
		return err
	}
	
	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		return fmt.Errorf("erreur suppression base de données %s: status %d", dbName, resp.StatusCode)
	}
	
	return nil
}

// GetDatabaseInfo récupère les informations d'une base de données
func (c *Client) GetDatabaseInfo(dbName string) (map[string]interface{}, error) {
	resp, err := c.Get("/" + dbName)
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("base de données non trouvée: %s", dbName)
	}
	
	var info map[string]interface{}
	if err := json.Unmarshal(resp.Body, &info); err != nil {
		return nil, fmt.Errorf("erreur parsing info: %v", err)
	}
	
	return info, nil
}

// BulkInsert insère plusieurs documents en une seule requête
func (c *Client) BulkInsert(dbName string, docs []map[string]interface{}) error {
	bulkData := map[string]interface{}{
		"docs": docs,
	}
	
	resp, err := c.Post("/"+dbName+"/_bulk_docs", bulkData)
	if err != nil {
		return fmt.Errorf("erreur bulk insert: %v", err)
	}
	
	if resp.StatusCode != 201 {
		return fmt.Errorf("erreur HTTP %d: %s", resp.StatusCode, string(resp.Body))
	}
	
	return nil
}

// GetDocument récupère un document par son ID
func (c *Client) GetDocument(dbName, docID string) (map[string]interface{}, error) {
	resp, err := c.Get(fmt.Sprintf("/%s/%s", dbName, docID))
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("document non trouvé: %s", docID)
	}
	
	var doc map[string]interface{}
	if err := json.Unmarshal(resp.Body, &doc); err != nil {
		return nil, fmt.Errorf("erreur parsing document: %v", err)
	}
	
	return doc, nil
}

// UpdateDocument met à jour un document
func (c *Client) UpdateDocument(dbName string, doc map[string]interface{}) error {
	docID, ok := doc["_id"].(string)
	if !ok {
		return fmt.Errorf("document sans _id")
	}
	
	resp, err := c.Put(fmt.Sprintf("/%s/%s", dbName, docID), doc)
	if err != nil {
		return err
	}
	
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return fmt.Errorf("erreur mise à jour document: status %d", resp.StatusCode)
	}
	
	return nil
}

// DeleteDocument supprime un document
func (c *Client) DeleteDocument(dbName, docID, rev string) error {
	resp, err := c.Delete(fmt.Sprintf("/%s/%s?rev=%s", dbName, docID, rev))
	if err != nil {
		return err
	}
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("erreur suppression document: status %d", resp.StatusCode)
	}
	
	return nil
}


// // pkg/client/client.go
// // Client HTTP pour communiquer avec CouchDB
// package client

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"strings"
// 	"time"
// )

// type Client struct {
// 	BaseURL    string
// 	Username   string
// 	Password   string
// 	HTTPClient *http.Client
// }

// type Response struct {
// 	StatusCode int
// 	Body       []byte
// 	Headers    http.Header
// }

// // New crée un nouveau client CouchDB
// func New(baseURL, username, password string) (*Client, error) {
// 	// Nettoyer l'URL
// 	baseURL = strings.TrimSuffix(baseURL, "/")
	
// 	// Si l'URL contient déjà les credentials, les extraire
// 	if strings.Contains(baseURL, "@") {
// 		parts := strings.Split(baseURL, "@")
// 		if len(parts) == 2 {
// 			authParts := strings.Split(strings.Replace(parts[0], "http://", "", 1), ":")
// 			if len(authParts) == 2 {
// 				username = authParts[0]
// 				password = authParts[1]
// 				baseURL = "http://" + parts[1]
// 			}
// 		}
// 	}
	
// 	return &Client{
// 		BaseURL:  baseURL,
// 		Username: username,
// 		Password: password,
// 		HTTPClient: &http.Client{
// 			Timeout: 30 * time.Second,
// 		},
// 	}, nil
// }

// // Get effectue une requête GET
// func (c *Client) Get(path string) (*Response, error) {
// 	url := c.buildURL(path)
	
// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("erreur création requête: %v", err)
// 	}
	
// 	c.addAuth(req)
// 	req.Header.Set("Content-Type", "application/json")
	
// 	resp, err := c.HTTPClient.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("erreur requête: %v", err)
// 	}
// 	defer resp.Body.Close()
	
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("erreur lecture réponse: %v", err)
// 	}
	
// 	return &Response{
// 		StatusCode: resp.StatusCode,
// 		Body:       body,
// 		Headers:    resp.Header,
// 	}, nil
// }

// // Post effectue une requête POST
// func (c *Client) Post(path string, data interface{}) (*Response, error) {
// 	url := c.buildURL(path)
	
// 	jsonData, err := json.Marshal(data)
// 	if err != nil {
// 		return nil, fmt.Errorf("erreur sérialisation JSON: %v", err)
// 	}
	
// 	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
// 	if err != nil {
// 		return nil, fmt.Errorf("erreur création requête: %v", err)
// 	}
	
// 	c.addAuth(req)
// 	req.Header.Set("Content-Type", "application/json")
	
// 	resp, err := c.HTTPClient.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("erreur requête: %v", err)
// 	}
// 	defer resp.Body.Close()
	
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("erreur lecture réponse: %v", err)
// 	}
	
// 	return &Response{
// 		StatusCode: resp.StatusCode,
// 		Body:       body,
// 		Headers:    resp.Header,
// 	}, nil
// }

// // Put effectue une requête PUT
// func (c *Client) Put(path string, data interface{}) (*Response, error) {
// 	url := c.buildURL(path)
	
// 	jsonData, err := json.Marshal(data)
// 	if err != nil {
// 		return nil, fmt.Errorf("erreur sérialisation JSON: %v", err)
// 	}
	
// 	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
// 	if err != nil {
// 		return nil, fmt.Errorf("erreur création requête: %v", err)
// 	}
	
// 	c.addAuth(req)
// 	req.Header.Set("Content-Type", "application/json")
	
// 	resp, err := c.HTTPClient.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("erreur requête: %v", err)
// 	}
// 	defer resp.Body.Close()
	
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("erreur lecture réponse: %v", err)
// 	}
	
// 	return &Response{
// 		StatusCode: resp.StatusCode,
// 		Body:       body,
// 		Headers:    resp.Header,
// 	}, nil
// }

// // Delete effectue une requête DELETE
// func (c *Client) Delete(path string) (*Response, error) {
// 	url := c.buildURL(path)
	
// 	req, err := http.NewRequest("DELETE", url, nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("erreur création requête: %v", err)
// 	}
	
// 	c.addAuth(req)
// 	req.Header.Set("Content-Type", "application/json")
	
// 	resp, err := c.HTTPClient.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("erreur requête: %v", err)
// 	}
// 	defer resp.Body.Close()
	
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("erreur lecture réponse: %v", err)
// 	}
	
// 	return &Response{
// 		StatusCode: resp.StatusCode,
// 		Body:       body,
// 		Headers:    resp.Header,
// 	}, nil
// }

// // buildURL construit l'URL complète
// func (c *Client) buildURL(path string) string {
// 	if !strings.HasPrefix(path, "/") {
// 		path = "/" + path
// 	}
// 	return c.BaseURL + path
// }

// // addAuth ajoute l'authentification à la requête
// func (c *Client) addAuth(req *http.Request) {
// 	if c.Username != "" && c.Password != "" {
// 		req.SetBasicAuth(c.Username, c.Password)
// 	}
// }

// // TestConnection teste la connexion au serveur CouchDB
// func (c *Client) TestConnection() error {
// 	resp, err := c.Get("/")
// 	if err != nil {
// 		return fmt.Errorf("impossible de se connecter à CouchDB: %v", err)
// 	}
	
// 	if resp.StatusCode != 200 {
// 		return fmt.Errorf("réponse inattendue de CouchDB: %d", resp.StatusCode)
// 	}
	
// 	return nil
// }

// // DatabaseExists vérifie si une base de données existe
// func (c *Client) DatabaseExists(dbName string) (bool, error) {
// 	resp, err := c.Get("/" + dbName)
// 	if err != nil {
// 		return false, err
// 	}
	
// 	return resp.StatusCode == 200, nil
// }

// // CreateDatabase crée une base de données
// func (c *Client) CreateDatabase(dbName string) error {
// 	resp, err := c.Put("/"+dbName, nil)
// 	if err != nil {
// 		return err
// 	}
	
// 	if resp.StatusCode != 201 && resp.StatusCode != 412 {
// 		return fmt.Errorf("erreur création base de données %s: status %d", dbName, resp.StatusCode)
// 	}
	
// 	return nil
// }

// // DeleteDatabase supprime une base de données
// func (c *Client) DeleteDatabase(dbName string) error {
// 	resp, err := c.Delete("/" + dbName)
// 	if err != nil {
// 		return err
// 	}
	
// 	if resp.StatusCode != 200 && resp.StatusCode != 404 {
// 		return fmt.Errorf("erreur suppression base de données %s: status %d", dbName, resp.StatusCode)
// 	}
	
// 	return nil
// }

// // GetDatabaseInfo récupère les informations d'une base de données
// func (c *Client) GetDatabaseInfo(dbName string) (map[string]interface{}, error) {
// 	resp, err := c.Get("/" + dbName)
// 	if err != nil {
// 		return nil, err
// 	}
	
// 	if resp.StatusCode != 200 {
// 		return nil, fmt.Errorf("base de données non trouvée: %s", dbName)
// 	}
	
// 	var info map[string]interface{}
// 	if err := json.Unmarshal(resp.Body, &info); err != nil {
// 		return nil, fmt.Errorf("erreur parsing info: %v", err)
// 	}
	
// 	return info, nil
// }

// // BulkInsert insère plusieurs documents en une seule requête
// func (c *Client) BulkInsert(dbName string, docs []map[string]interface{}) error {
// 	bulkData := map[string]interface{}{
// 		"docs": docs,
// 	}
	
// 	resp, err := c.Post("/"+dbName+"/_bulk_docs", bulkData)
// 	if err != nil {
// 		return fmt.Errorf("erreur bulk insert: %v", err)
// 	}
	
// 	if resp.StatusCode != 201 {
// 		return fmt.Errorf("erreur HTTP %d: %s", resp.StatusCode, string(resp.Body))
// 	}
	
// 	return nil
// }

// // GetDocument récupère un document par son ID
// func (c *Client) GetDocument(dbName, docID string) (map[string]interface{}, error) {
// 	resp, err := c.Get(fmt.Sprintf("/%s/%s", dbName, docID))
// 	if err != nil {
// 		return nil, err
// 	}
	
// 	if resp.StatusCode != 200 {
// 		return nil, fmt.Errorf("document non trouvé: %s", docID)
// 	}
	
// 	var doc map[string]interface{}
// 	if err := json.Unmarshal(resp.Body, &doc); err != nil {
// 		return nil, fmt.Errorf("erreur parsing document: %v", err)
// 	}
	
// 	return doc, nil
// }

// // UpdateDocument met à jour un document
// func (c *Client) UpdateDocument(dbName string, doc map[string]interface{}) error {
// 	docID, ok := doc["_id"].(string)
// 	if !ok {
// 		return fmt.Errorf("document sans _id")
// 	}
	
// 	resp, err := c.Put(fmt.Sprintf("/%s/%s", dbName, docID), doc)
// 	if err != nil {
// 		return err
// 	}
	
// 	if resp.StatusCode != 201 && resp.StatusCode != 200 {
// 		return fmt.Errorf("erreur mise à jour document: status %d", resp.StatusCode)
// 	}
	
// 	return nil
// }

// // DeleteDocument supprime un document
// func (c *Client) DeleteDocument(dbName, docID, rev string) error {
// 	resp, err := c.Delete(fmt.Sprintf("/%s/%s?rev=%s", dbName, docID, rev))
// 	if err != nil {
// 		return err
// 	}
	
// 	if resp.StatusCode != 200 {
// 		return fmt.Errorf("erreur suppression document: status %d", resp.StatusCode)
// 	}
	
// 	return nil
// }