// cmd/monitor/main.go
// Programme de surveillance du cluster CouchDB
package main

import (
	"flag"
	"log"
	"time"
	"strings"

	"couchdb-tp/pkg/cluster"
	"couchdb-tp/pkg/monitor"
)

func main() {
	var (
		interval   = flag.Int("interval", 10, "Intervalle de surveillance en secondes")
		continuous = flag.Bool("continuous", false, "Surveillance continue")
		database   = flag.String("database", "ecommerce_orders", "Base de donnees a surveiller")
		verbose    = flag.Bool("verbose", false, "Affichage detaille")
	)
	flag.Parse()

	// Configuration du format des logs
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Surveillance du Cluster CouchDB")
	log.Println("===============================")

	// Creation de la configuration du cluster
	config := cluster.NewClusterConfig()
	mon := monitor.New(config)

	if *continuous {
		log.Printf("Demarrage de la surveillance continue (intervalle: %ds)...", *interval)
		log.Println("Appuyez sur Ctrl+C pour arreter")
		log.Println("")

		// Boucle de surveillance continue
		for {
			showTimestamp()
			
			log.Println("=== VERIFICATION SANTE DU CLUSTER ===")
			mon.CheckClusterHealth()
			
			log.Println("")
			log.Printf("=== COHERENCE DES DONNEES (%s) ===", *database)
			mon.CheckDataConsistency(*database)
			
			log.Println("")
			log.Println("=== STATUT DE LA REPLICATION ===")
			mon.CheckReplicationStatus()
			
			if *verbose {
				log.Println("")
				log.Println("=== METRIQUES DETAILLEES ===")
				mon.ShowDetailedMetrics()
			}

			log.Println("")
			log.Printf("Prochaine verification dans %d secondes...", *interval)
			log.Println(strings.Repeat("=", 50))
			log.Println("")
			
			time.Sleep(time.Duration(*interval) * time.Second)
		}
	} else {
		// Surveillance ponctuelle
		showTimestamp()
		
		log.Println("=== VERIFICATION SANTE DU CLUSTER ===")
		mon.CheckClusterHealth()
		
		log.Println("")
		log.Printf("=== COHERENCE DES DONNEES (%s) ===", *database)
		mon.CheckDataConsistency(*database)
		
		log.Println("")
		log.Println("=== STATUT DE LA REPLICATION ===")
		mon.CheckReplicationStatus()
		
		if *verbose {
			log.Println("")
			log.Println("=== METRIQUES DETAILLEES ===")
			mon.ShowDetailedMetrics()
		}
		
		log.Println("")
		showPostMonitoringInfo(*database)
	}
}

// showTimestamp affiche l'horodatage de la verification
func showTimestamp() {
	log.Printf("Verification du cluster - %s", time.Now().Format("2006-01-02 15:04:05"))
}

// showPostMonitoringInfo affiche les informations post-surveillance
func showPostMonitoringInfo(database string) {
	log.Println("Informations de Surveillance")
	log.Println("============================")
	
	log.Println("Commandes utiles:")
	log.Println("  Surveillance continue: ./bin/monitor -continuous -interval 30")
	log.Println("  Mode verbose: ./bin/monitor -verbose")
	log.Printf("  Autre base: ./bin/monitor -database %s", "ecommerce_products")
	
	log.Println("")
	log.Println("Surveillance manuelle:")
	log.Println("  Status cluster: ./bin/setup -mode verify")
	log.Println("  Interfaces Fauxton:")
	log.Println("    http://localhost:5987/_utils (Amerique du Nord 1)")
	log.Println("    http://localhost:5988/_utils (Amerique du Nord 2)")
	log.Println("    http://localhost:5989/_utils (Europe 1)")
	log.Println("    http://localhost:5990/_utils (Asie Pacifique 1)")
	
	log.Println("")
	log.Println("APIs de surveillance:")
	log.Println("  curl http://admin:ecommerce2024@localhost:5987/_up")
	log.Println("  curl http://admin:ecommerce2024@localhost:5987/ecommerce_orders")
	log.Println("  curl http://admin:ecommerce2024@localhost:5987/_replicator/_all_docs")
	
	log.Println("")
	log.Println("Indicateurs a surveiller:")
	log.Println("  - Tous les noeuds sont actifs")
	log.Println("  - Coherence des donnees entre noeuds")
	log.Println("  - Replications actives")
	log.Println("  - Absence d'erreurs de replication")
}