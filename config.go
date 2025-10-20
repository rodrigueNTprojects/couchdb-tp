// config.go
// Configuration de la réplication distribuée pour le cluster CouchDB
// À intégrer dans cmd/setup/main.go

package main

import (
    "fmt"
    "log"
    "strings"
    "time"
    "couchdb-tp/pkg/client"
)

type PartialClusterConfig struct {
    Nodes map[string]NodeInfo
    Username string
    Password string
}

type NodeInfo struct {
    URL         string
    InternalURL string
    Description string
    Region      string
}

func NewPartialClusterConfig() *PartialClusterConfig {
    return &PartialClusterConfig{
        Nodes: map[string]NodeInfo{
            "NA1": {URL: "http://localhost:5987", InternalURL: "http://192.168.100.10:5984", Description: "Amerique du Nord 1", Region: "north_america"},
            "NA2": {URL: "http://localhost:5988", InternalURL: "http://192.168.100.11:5984", Description: "Amerique du Nord 2", Region: "north_america"},
            "EU1": {URL: "http://localhost:5989", InternalURL: "http://192.168.100.12:5984", Description: "Europe 1", Region: "europe"},
            "AP1": {URL: "http://localhost:5990", InternalURL: "http://192.168.100.13:5984", Description: "Asie Pacifique 1", Region: "asia_pacific"},
        },
        Username: "admin",
        Password: "ecommerce2024",
    }
}

// FOURNI : Configuration de la réplication bidirectionnelle partielle
func (c *PartialClusterConfig) SetupPartialReplication() error {
    log.Println("Configuration partielle de la replication pour registres distribues...")
    
    // Bases de registres à répliquer
    ledgerDatabases := []string{"ecommerce_orders", "ecommerce_products", "ecommerce_sellers", "ecommerce_leads"}
    
    // FOURNI : Réplications entre nœuds Amérique du Nord seulement
    partialReplications := []struct {
        Source string
        Target string
        Name   string
    }{
        // Réplication bidirectionnelle NA1 <-> NA2 (FOURNI)
        {Source: "NA1", Target: "NA2", Name: "na1_to_na2"},
        {Source: "NA2", Target: "NA1", Name: "na2_to_na1"},
		
        
        // TODO: COMPLETEZ LES REPLICATIONS MANQUANTES
       
    
    }
    
    // Code de configuration de la réplication (fourni)
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
        sourceAuthURL := strings.Replace(sourceNode.InternalURL, "http://", fmt.Sprintf("http://%s:%s@", c.Username, c.Password), 1)
        targetAuthURL := strings.Replace(targetNode.InternalURL, "http://", fmt.Sprintf("http://%s:%s@", c.Username, c.Password), 1)
        
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