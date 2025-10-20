// pkg/monitor/monitor.go
// Package de surveillance du cluster CouchDB
package monitor

import (
	"encoding/json"
	"log"
	"strings"

	"couchdb-tp/pkg/client"
	"couchdb-tp/pkg/cluster"
)

// Monitor represente un moniteur de cluster CouchDB
type Monitor struct {
	config *cluster.ClusterConfig // Configuration du cluster
}

// DatabaseStats contient les statistiques d'une base de donnees
type DatabaseStats struct {
	DocCount int    `json:"doc_count"` // Nombre de documents
	DiskSize int64  `json:"disk_size"` // Taille sur disque en octets
	DataSize int64  `json:"data_size"` // Taille des donnees en octets
	Status   string // Statut de la base (OK, VIDE, ERREUR)
}

// NodeHealth contient l'etat de sante d'un noeud
type NodeHealth struct {
	Name        string        // Nom du noeud
	Description string        // Description du noeud
	IsActive    bool          // Noeud actif ou non
	Version     string        // Version CouchDB
	Error       string        // Message d'erreur si applicable
}

// New cree un nouveau moniteur pour le cluster specifie
func New(config *cluster.ClusterConfig) *Monitor {
	return &Monitor{
		config: config,
	}
}

// CheckClusterHealth verifie la sante de tous les noeuds du cluster
func (m *Monitor) CheckClusterHealth() {
	log.Println("Verification de la sante du cluster...")

	var healthyNodes, unhealthyNodes int

	for nodeName, nodeInfo := range m.config.Nodes {
		health := m.checkNodeHealth(nodeName, nodeInfo)
		
		if health.IsActive {
			log.Printf("%s: ACTIF (v%s)", health.Description, health.Version)
			healthyNodes++
		} else {
			log.Printf("%s: INACTIF (%s)", health.Description, health.Error)
			unhealthyNodes++
		}
	}

	log.Printf("Resume: %d noeuds actifs, %d noeuds inactifs", healthyNodes, unhealthyNodes)
	
	if unhealthyNodes > 0 {
		log.Println("ATTENTION: Des noeuds sont inactifs - verifiez les conteneurs Docker")
	}
}

// checkNodeHealth verifie la sante d'un noeud specifique
func (m *Monitor) checkNodeHealth(nodeName string, nodeInfo cluster.NodeInfo) NodeHealth {
	nodeClient, err := client.New(nodeInfo.URL, m.config.Username, m.config.Password)
	if err != nil {
		return NodeHealth{
			Name:        nodeName,
			Description: nodeInfo.Description,
			IsActive:    false,
			Error:       "impossible de creer le client",
		}
	}

	resp, err := nodeClient.Get("/")
	if err != nil {
		return NodeHealth{
			Name:        nodeName,
			Description: nodeInfo.Description,
			IsActive:    false,
			Error:       "connexion echouee",
		}
	}

	if resp.StatusCode != 200 {
		return NodeHealth{
			Name:        nodeName,
			Description: nodeInfo.Description,
			IsActive:    false,
			Error:       "statut HTTP invalide",
		}
	}

	// Extraction de la version CouchDB
	var info map[string]interface{}
	json.Unmarshal(resp.Body, &info)
	version := "inconnue"
	if v, ok := info["version"]; ok {
		if vStr, ok := v.(string); ok {
			version = vStr
		}
	}

	return NodeHealth{
		Name:        nodeName,
		Description: nodeInfo.Description,
		IsActive:    true,
		Version:     version,
	}
}

// CheckDataConsistency verifie la coherence des donnees entre les noeuds
func (m *Monitor) CheckDataConsistency(database string) {
	log.Printf("Verification de la coherence des donnees pour '%s'...", database)

	var stats []DatabaseStats
	var nodeNames []string

	// Collecte des statistiques de chaque noeud
	for nodeName, nodeInfo := range m.config.Nodes {
		stat := m.getDatabaseStats(nodeName, nodeInfo, database)
		stats = append(stats, stat)
		nodeNames = append(nodeNames, nodeName)
		
		log.Printf("Noeud %s: %d documents (%s)", nodeName, stat.DocCount, stat.Status)
	}

	// Analyse de la coherence
	m.analyzeConsistency(stats, nodeNames, database)
}

// getDatabaseStats recupere les statistiques d'une base de donnees sur un noeud
func (m *Monitor) getDatabaseStats(nodeName string, nodeInfo cluster.NodeInfo, database string) DatabaseStats {
	nodeClient, err := client.New(nodeInfo.URL, m.config.Username, m.config.Password)
	if err != nil {
		return DatabaseStats{Status: "ERREUR_CONNEXION"}
	}

	resp, err := nodeClient.Get("/" + database)
	if err != nil {
		return DatabaseStats{Status: "ERREUR_REQUETE"}
	}

	if resp.StatusCode == 404 {
		return DatabaseStats{Status: "INEXISTANTE"}
	}

	if resp.StatusCode != 200 {
		return DatabaseStats{Status: "ERREUR_HTTP"}
	}

	var dbInfo DatabaseStats
	if err := json.Unmarshal(resp.Body, &dbInfo); err != nil {
		return DatabaseStats{Status: "ERREUR_ANALYSE"}
	}

	// Determination du statut
	if dbInfo.DocCount > 0 {
		dbInfo.Status = "OK"
	} else {
		dbInfo.Status = "VIDE"
	}

	return dbInfo
}

// analyzeConsistency analyse la coherence des donnees entre les noeuds
func (m *Monitor) analyzeConsistency(stats []DatabaseStats, nodeNames []string, database string) {
	if len(stats) == 0 {
		log.Println("Aucune statistique a analyser")
		return
	}

	// Verification de la coherence des comptes de documents
	var counts []int
	for _, stat := range stats {
		counts = append(counts, stat.DocCount)
	}

	// Recherche du maximum et minimum
	maxCount := counts[0]
	minCount := counts[0]
	for _, count := range counts {
		if count > maxCount {
			maxCount = count
		}
		if count < minCount {
			minCount = count
		}
	}

	// Evaluation de la coherence
	if maxCount == minCount && maxCount > 0 {
		log.Printf("COHERENT: Tous les noeuds ont %d documents", maxCount)
	} else if maxCount == minCount && maxCount == 0 {
		log.Printf("VIDE: Tous les noeuds sont vides pour '%s'", database)
	} else {
		log.Printf("INCOHERENT: Ecart de %d a %d documents", minCount, maxCount)
		log.Println("La replication est probablement en cours...")
		
		// Affichage des details par noeud
		for i, stat := range stats {
			if stat.DocCount == maxCount {
				log.Printf("  %s: %d documents (a jour)", nodeNames[i], stat.DocCount)
			} else {
				log.Printf("  %s: %d documents (en retard)", nodeNames[i], stat.DocCount)
			}
		}
	}
}

// CheckReplicationStatus verifie le statut de la replication
func (m *Monitor) CheckReplicationStatus() {
	log.Println("Verification du statut de la replication...")

	totalReplications := 0
	activeReplications := 0
	errorReplications := 0

	for nodeName, nodeInfo := range m.config.Nodes {
		nodeReplications := m.getNodeReplicationStats(nodeName, nodeInfo)
		totalReplications += nodeReplications.Total
		activeReplications += nodeReplications.Active
		errorReplications += nodeReplications.Errors
	}

	log.Printf("Resume global:")
	log.Printf("  Total replications configurees: %d", totalReplications)
	log.Printf("  Replications actives: %d", activeReplications)
	log.Printf("  Replications en erreur: %d", errorReplications)

	if errorReplications > 0 {
		log.Println("ATTENTION: Des erreurs de replication detectees")
	} else if activeReplications > 0 {
		log.Println("STATUS: Replication fonctionnelle")
	} else {
		log.Println("AVERTISSEMENT: Aucune replication active detectee")
	}
}

// ReplicationStats contient les statistiques de replication d'un noeud
type ReplicationStats struct {
	Total  int // Nombre total de documents de replication
	Active int // Nombre de replications actives
	Errors int // Nombre de replications en erreur
}

// getNodeReplicationStats recupere les statistiques de replication d'un noeud
func (m *Monitor) getNodeReplicationStats(nodeName string, nodeInfo cluster.NodeInfo) ReplicationStats {
	nodeClient, err := client.New(nodeInfo.URL, m.config.Username, m.config.Password)
	if err != nil {
		log.Printf("Erreur connexion %s pour replication: %v", nodeName, err)
		return ReplicationStats{}
	}

	resp, err := nodeClient.Get("/_replicator/_all_docs")
	if err != nil {
		log.Printf("Erreur lecture replicator %s: %v", nodeName, err)
		return ReplicationStats{}
	}

	var replicatorResp struct {
		TotalRows int `json:"total_rows"`
		Rows      []struct {
			ID string `json:"id"`
		} `json:"rows"`
	}

	if err := json.Unmarshal(resp.Body, &replicatorResp); err != nil {
		log.Printf("Erreur analyse reponse replicator %s: %v", nodeName, err)
		return ReplicationStats{}
	}

	stats := ReplicationStats{Total: replicatorResp.TotalRows}

	// Verification par base de donnees
	databases := []string{"ecommerce_orders", "ecommerce_products", "ecommerce_sellers", "ecommerce_leads"}
	log.Printf("%s:", nodeInfo.Description)
	log.Printf("  Documents de replication: %d", stats.Total)

	for _, dbName := range databases {
		count := 0
		for _, row := range replicatorResp.Rows {
			if strings.Contains(row.ID, dbName) {
				count++
			}
		}

		if count > 0 {
			log.Printf("    %s: %d replications", dbName, count)
			stats.Active += count
		} else {
			log.Printf("    %s: AUCUNE replication", dbName)
		}
	}

	return stats
}

// ShowDetailedMetrics affiche des metriques detaillees du cluster
func (m *Monitor) ShowDetailedMetrics() {
	log.Println("Metriques detaillees du cluster...")

	for nodeName, nodeInfo := range m.config.Nodes {
		log.Printf("\n%s (%s):", nodeInfo.Description, nodeName)
		m.showNodeMetrics(nodeName, nodeInfo)
	}
}

// showNodeMetrics affiche les metriques detaillees d'un noeud
func (m *Monitor) showNodeMetrics(nodeName string, nodeInfo cluster.NodeInfo) {
	nodeClient, err := client.New(nodeInfo.URL, m.config.Username, m.config.Password)
	if err != nil {
		log.Printf("  Impossible de recuperer les metriques: %v", err)
		return
	}

	// Statistiques generales
	resp, err := nodeClient.Get("/_stats")
	if err != nil {
		log.Printf("  Erreur recuperation statistiques: %v", err)
		return
	}

	// Pour l'instant, affichage simple - peut etre etendu
	log.Printf("  Statistiques disponibles: Oui")
	log.Printf("  Taille reponse stats: %d octets", len(resp.Body))

	// Verification de l'espace disque par base
	databases := []string{"ecommerce_orders", "ecommerce_products", "ecommerce_sellers", "ecommerce_leads"}
	for _, dbName := range databases {
		stats := m.getDatabaseStats(nodeName, nodeInfo, dbName)
		if stats.Status == "OK" {
			diskSizeMB := float64(stats.DiskSize) / (1024 * 1024)
			dataSizeMB := float64(stats.DataSize) / (1024 * 1024)
			log.Printf("  %s: %.2f MB disque, %.2f MB donnees", dbName, diskSizeMB, dataSizeMB)
		}
	}
}