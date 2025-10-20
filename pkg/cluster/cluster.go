// pkg/cluster/cluster.go
// Gestion du cluster CouchDB distribué
package cluster

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"couchdb-tp/pkg/client"
)

type ClusterConfig struct {
	Nodes    map[string]NodeInfo
	Username string
	Password string
}

type NodeInfo struct {
	URL         string
	InternalURL string
	Description string
	Region      string
}

func NewClusterConfig() *ClusterConfig {
	return &ClusterConfig{
		Nodes: map[string]NodeInfo{
			"NA1": {
				URL:         "http://localhost:5987",
				InternalURL: "http://192.168.100.10:5984",
				Description: "Amerique du Nord 1",
				Region:      "north_america",
			},
			"NA2": {
				URL:         "http://localhost:5988",
				InternalURL: "http://192.168.100.11:5984",
				Description: "Amerique du Nord 2",
				Region:      "north_america",
			},
			"EU1": {
				URL:         "http://localhost:5989",
				InternalURL: "http://192.168.100.12:5984",
				Description: "Europe 1",
				Region:      "europe",
			},
			"AP1": {
				URL:         "http://localhost:5990",
				InternalURL: "http://192.168.100.13:5984",
				Description: "Asie Pacifique 1",
				Region:      "asia_pacific",
			},
		},
		Username: "admin",
		Password: "ecommerce2024",
	}
}

// SetupReplication configure la réplication bidirectionnelle complète
func (c *ClusterConfig) SetupReplication() error {
	log.Println("Configuration de la replication pour registres distribues...")

	// Bases de registres à répliquer
	ledgerDatabases := []string{
		"ecommerce_orders",
		"ecommerce_products",
		"ecommerce_sellers",
		"ecommerce_leads",
	}

	// SOLUTION COMPLÈTE : Toutes les 12 réplications bidirectionnelles
	partialReplications := []struct {
		Source string
		Target string
		Name   string
	}{
		// NA1 <-> NA2
		{Source: "NA1", Target: "NA2", Name: "na1_to_na2"},
		{Source: "NA2", Target: "NA1", Name: "na2_to_na1"},

		// NA1 <-> EU1
		{Source: "NA1", Target: "EU1", Name: "na1_to_eu1"},
		{Source: "EU1", Target: "NA1", Name: "eu1_to_na1"},

		// NA1 <-> AP1
		{Source: "NA1", Target: "AP1", Name: "na1_to_ap1"},
		{Source: "AP1", Target: "NA1", Name: "ap1_to_na1"},

		// NA2 <-> EU1
		{Source: "NA2", Target: "EU1", Name: "na2_to_eu1"},
		{Source: "EU1", Target: "NA2", Name: "eu1_to_na2"},

		// NA2 <-> AP1
		{Source: "NA2", Target: "AP1", Name: "na2_to_ap1"},
		{Source: "AP1", Target: "NA2", Name: "ap1_to_na2"},

		// EU1 <-> AP1
		{Source: "EU1", Target: "AP1", Name: "eu1_to_ap1"},
		{Source: "AP1", Target: "EU1", Name: "ap1_to_eu1"},
	}

	// Code de configuration de la réplication
	for _, repl := range partialReplications {
		sourceNode := c.Nodes[repl.Source]
		targetNode := c.Nodes[repl.Target]

		log.Printf("Configuration replication: %s -> %s", sourceNode.Description, targetNode.Description)

		sourceClient, err := client.New(sourceNode.URL, c.Username, c.Password)
		if err != nil {
			log.Printf("Erreur creation client source %s: %v", repl.Source, err)
			continue
		}

		// URLs authentifiées pour la réplication
		sourceAuthURL := strings.Replace(sourceNode.InternalURL, "http://",
			fmt.Sprintf("http://%s:%s@", c.Username, c.Password), 1)
		targetAuthURL := strings.Replace(targetNode.InternalURL, "http://",
			fmt.Sprintf("http://%s:%s@", c.Username, c.Password), 1)

		// Configuration pour chaque registre
		for _, dbName := range ledgerDatabases {
			replicationID := fmt.Sprintf("ledger_replication_%s_%s", repl.Name, dbName)

			// Suppression ancienne réplication
			sourceClient.Delete("/_replicator/" + replicationID)
			time.Sleep(1 * time.Second)

			// Document de réplication pour registre
			replicationDoc := map[string]interface{}{
				"_id":           replicationID,
				"source":        sourceAuthURL + "/" + dbName,
				"target":        targetAuthURL + "/" + dbName,
				"continuous":    true,
				"create_target": true,
				"owner":         "admin",
				"ledger_type":   "distributed_registry",
			}

			resp, err := sourceClient.Put("/_replicator/"+replicationID, replicationDoc)
			if err != nil {
				log.Printf("    Avertissement %s: %v", dbName, err)
			} else if resp.StatusCode == 201 {
				log.Printf("    Replication registre %s configuree", dbName)
			}
		}
	}

	return nil
}

// WaitForNodes attend que tous les nœuds soient disponibles
func (c *ClusterConfig) WaitForNodes() error {
	maxRetries := 30
	retryDelay := 2 * time.Second

	for _, nodeInfo := range c.Nodes {
		log.Printf("Attente disponibilite %s...", nodeInfo.Description)

		nodeClient, err := client.New(nodeInfo.URL, c.Username, c.Password)
		if err != nil {
			return fmt.Errorf("erreur creation client %s: %v", nodeInfo.Description, err)
		}

		for i := 0; i < maxRetries; i++ {
			err := nodeClient.TestConnection()
			if err == nil {
				log.Printf("  %s disponible", nodeInfo.Description)
				break
			}

			if i == maxRetries-1 {
				return fmt.Errorf("timeout attente %s", nodeInfo.Description)
			}

			time.Sleep(retryDelay)
		}
	}

	return nil
}

// CreateSystemDatabases crée les bases de données système
func (c *ClusterConfig) CreateSystemDatabases() error {
	systemDatabases := []string{"_users", "_replicator"}

	for _, nodeInfo := range c.Nodes {
		log.Printf("Creation bases systeme sur %s...", nodeInfo.Description)

		nodeClient, err := client.New(nodeInfo.URL, c.Username, c.Password)
		if err != nil {
			return fmt.Errorf("erreur client %s: %v", nodeInfo.Description, err)
		}

		for _, dbName := range systemDatabases {
			exists, _ := nodeClient.DatabaseExists(dbName)
			if !exists {
				if err := nodeClient.CreateDatabase(dbName); err != nil {
					log.Printf("  Erreur creation %s: %v", dbName, err)
				} else {
					log.Printf("  Base %s creee", dbName)
				}
			} else {
				log.Printf("  Base %s existe deja", dbName)
			}
		}
	}

	return nil
}

// CreateEcommerceDatabases crée les bases de données e-commerce
func (c *ClusterConfig) CreateEcommerceDatabases() error {
	ecommerceDatabases := []string{
		"ecommerce_orders",
		"ecommerce_products",
		"ecommerce_sellers",
		"ecommerce_leads",
	}

	for _, nodeInfo := range c.Nodes {
		log.Printf("Creation bases e-commerce sur %s...", nodeInfo.Description)

		nodeClient, err := client.New(nodeInfo.URL, c.Username, c.Password)
		if err != nil {
			return fmt.Errorf("erreur client %s: %v", nodeInfo.Description, err)
		}

		for _, dbName := range ecommerceDatabases {
			exists, _ := nodeClient.DatabaseExists(dbName)
			if !exists {
				if err := nodeClient.CreateDatabase(dbName); err != nil {
					log.Printf("  Erreur creation %s: %v", dbName, err)
				} else {
					log.Printf("  Base %s creee", dbName)
				}
			} else {
				log.Printf("  Base %s existe deja", dbName)
			}
		}
	}

	return nil
}

// GetStatus vérifie et affiche le statut du cluster
func (c *ClusterConfig) GetStatus() error {
	log.Println("")

	for _, nodeInfo := range c.Nodes {
		log.Printf("%s:", nodeInfo.Description)

		nodeClient, err := client.New(nodeInfo.URL, c.Username, c.Password)
		if err != nil {
			log.Printf("  Erreur connexion: %v", err)
			continue
		}

		// Tester connexion
		if err := nodeClient.TestConnection(); err != nil {
			log.Printf("  Status: OFFLINE")
			continue
		}
		log.Printf("  Status: ONLINE")

		// Compter les bases
		systemBases := 0
		ecommerceBases := 0

		databases := []string{
			"_users", "_replicator",
			"ecommerce_orders", "ecommerce_products",
			"ecommerce_sellers", "ecommerce_leads",
		}

		for _, db := range databases {
			exists, _ := nodeClient.DatabaseExists(db)
			if exists {
				if strings.HasPrefix(db, "_") {
					systemBases++
				} else {
					ecommerceBases++
				}
			}
		}

		log.Printf("  Bases systeme: %d", systemBases)
		log.Printf("  Bases e-commerce: %d", ecommerceBases)

		// Détails des bases e-commerce
		for _, db := range []string{"ecommerce_orders", "ecommerce_products", "ecommerce_sellers", "ecommerce_leads"} {
			info, err := nodeClient.GetDatabaseInfo(db)
			if err == nil {
				docCount := int(info["doc_count"].(float64))
				log.Printf("    %s: %d documents", db, docCount)
			} else {
				log.Printf("    %s: non disponible", db)
			}
		}

		log.Println("")
	}

	// Statut de la réplication
	log.Println("Statut de la replication:")
	log.Println("")

	for _, nodeInfo := range c.Nodes {
		log.Printf("%s:", nodeInfo.Description)

		nodeClient, err := client.New(nodeInfo.URL, c.Username, c.Password)
		if err != nil {
			continue
		}

		// Compter documents de réplication
		resp, err := nodeClient.Get("/_replicator/_all_docs")
		if err != nil {
			log.Printf("  Erreur lecture replications: %v", err)
			continue
		}

		var allDocs map[string]interface{}
		if err := json.Unmarshal(resp.Body, &allDocs); err != nil {
			continue
		}

		rows, ok := allDocs["rows"].([]interface{})
		if !ok {
			continue
		}

		// Compter par base
		replicationCount := make(map[string]int)
		for _, row := range rows {
			rowMap := row.(map[string]interface{})
			docID := rowMap["id"].(string)

			if strings.HasPrefix(docID, "ledger_replication_") {
				// Extraire le nom de la base
				parts := strings.Split(docID, "_")
				if len(parts) >= 4 {
					dbName := strings.Join(parts[3:], "_")
					replicationCount[dbName]++
				}
			}
		}

		totalReplications := 0
		for _, count := range replicationCount {
			totalReplications += count
		}

		log.Printf("  Documents de replication: %d", totalReplications)

		for dbName, count := range replicationCount {
			log.Printf("    %s: %d replications", dbName, count)
		}

		log.Println("")
	}

	return nil
}

// VerifyReplication vérifie que la réplication fonctionne
func (c *ClusterConfig) VerifyReplication() error {
	log.Println("Verification de la coherence de la replication...")

	databases := []string{
		"ecommerce_orders",
		"ecommerce_products",
		"ecommerce_sellers",
		"ecommerce_leads",
	}

	for _, dbName := range databases {
		log.Printf("Base: %s", dbName)

		docCounts := make(map[string]int)

		// Récupérer le nombre de documents sur chaque nœud
		for nodeName, nodeInfo := range c.Nodes {
			nodeClient, err := client.New(nodeInfo.URL, c.Username, c.Password)
			if err != nil {
				log.Printf("  %s: erreur connexion", nodeInfo.Description)
				continue
			}

			info, err := nodeClient.GetDatabaseInfo(dbName)
			if err != nil {
				log.Printf("  %s: erreur lecture info", nodeInfo.Description)
				continue
			}

			docCount := int(info["doc_count"].(float64))
			docCounts[nodeName] = docCount
			log.Printf("  %s: %d documents", nodeInfo.Description, docCount)
		}

		// Vérifier cohérence
		if len(docCounts) > 0 {
			firstCount := -1
			coherent := true

			for _, count := range docCounts {
				if firstCount == -1 {
					firstCount = count
				} else if count != firstCount {
					coherent = false
					break
				}
			}

			if coherent {
				log.Printf("  ✓ Replication coherente (%d documents)", firstCount)
			} else {
				log.Printf("  ✗ Replication incoherente - synchronisation en cours")
			}
		}

		log.Println("")
	}

	return nil
}