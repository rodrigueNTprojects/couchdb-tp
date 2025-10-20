// cmd/cleanup/main.go
// Programme de nettoyage de l'environnement CouchDB
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"couchdb-tp/pkg/client"
	"couchdb-tp/pkg/cluster"
)

func main() {
	var (
		full      = flag.Bool("full", false, "Nettoyage complet (conteneurs + donnees)")
		databases = flag.Bool("databases", false, "Nettoyer seulement les bases de donnees")
		services  = flag.Bool("services", false, "Arreter seulement les services")
		force     = flag.Bool("force", false, "Forcer le nettoyage sans confirmation")
		verbose   = flag.Bool("verbose", false, "Affichage detaille")
	)
	flag.Parse()

	// Configuration du format des logs
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Nettoyage de l'Environnement CouchDB")
	log.Println("====================================")

	// Affichage de ce qui sera nettoye
	showCleanupPlan(*full, *databases, *services)

	// Execution des operations de nettoyage selon les options
	if *full || *databases {
		cleanupDatabases(*force, *verbose)
	}

	if *full || *services {
		stopServices(*force, *verbose)
	}

	if *full {
		cleanupContainers(*force, *verbose)
		cleanupTempFiles(*verbose)
	}

	log.Println("")
	log.Println("Nettoyage termine!")
	showPostCleanupInfo(*full)
}

// showCleanupPlan affiche ce qui sera nettoye
func showCleanupPlan(full, databases, services bool) {
	log.Println("Plan de nettoyage:")
	
	if full {
		log.Println("  - Suppression des bases de donnees e-commerce")
		log.Println("  - Arret des services CouchDB")
		log.Println("  - Suppression des conteneurs et volumes")
		log.Println("  - Nettoyage des fichiers temporaires")
	} else {
		if databases {
			log.Println("  - Suppression des bases de donnees e-commerce")
		}
		if services {
			log.Println("  - Arret des services CouchDB")
		}
	}
	log.Println("")
}

// cleanupDatabases supprime les bases de donnees e-commerce
func cleanupDatabases(force, verbose bool) {
	log.Println("=== NETTOYAGE DES BASES DE DONNEES ===")

	if !force {
		fmt.Print("Supprimer toutes les bases de donnees e-commerce? (oui/non): ")
		var response string
		fmt.Scanln(&response)
		if response != "oui" {
			log.Println("Nettoyage des bases de donnees ignore")
			return
		}
	}

	// Configuration du cluster
	config := cluster.NewClusterConfig()
	databases := []string{"ecommerce_orders", "ecommerce_products", "ecommerce_sellers", "ecommerce_leads"}

	// Suppression sur chaque noeud
	for _, node := range config.Nodes {
		log.Printf("Nettoyage des bases sur %s...", node.Description)

		couchClient, err := client.New(node.URL, config.Username, config.Password)
		if err != nil {
			log.Printf("Impossible de se connecter a %s: %v", node.Name, err)
			continue
		}

		// Suppression de chaque base de donnees
		for _, dbName := range databases {
			resp, err := couchClient.Delete("/" + dbName)
			if err != nil {
				if verbose {
					log.Printf("  Erreur suppression %s: %v", dbName, err)
				}
			} else if resp.StatusCode == 200 {
				log.Printf("  Base %s supprimee", dbName)
			} else if resp.StatusCode == 404 {
				if verbose {
					log.Printf("  Base %s n'existe pas", dbName)
				}
			} else {
				log.Printf("  Avertissement %s: statut %d", dbName, resp.StatusCode)
			}
		}
	}

	log.Println("Nettoyage des bases de donnees termine")
}

// stopServices arrete les services CouchDB
func stopServices(force, verbose bool) {
	log.Println("=== ARRET DES SERVICES COUCHDB ===")

	if !force {
		fmt.Print("Arreter les services CouchDB? (oui/non): ")
		var response string
		fmt.Scanln(&response)
		if response != "oui" {
			log.Println("Arret des services ignore")
			return
		}
	}

	log.Println("Arret des conteneurs CouchDB...")
	
	cmd := exec.Command("docker-compose", "-f", "docker/docker-compose-cluster.yml", "stop")
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	
	err := cmd.Run()
	if err != nil {
		log.Printf("Erreur arret des services: %v", err)
	} else {
		log.Println("Services CouchDB arretes")
	}
}

// cleanupContainers supprime les conteneurs et volumes
func cleanupContainers(force, verbose bool) {
	log.Println("=== SUPPRESSION DES CONTENEURS ET VOLUMES ===")

	if !force {
		fmt.Print("Supprimer les conteneurs et volumes? (oui/non): ")
		var response string
		fmt.Scanln(&response)
		if response != "oui" {
			log.Println("Suppression des conteneurs ignoree")
			return
		}
	}

	log.Println("Suppression des conteneurs et volumes...")
	
	cmd := exec.Command("docker-compose", "-f", "docker/docker-compose-cluster.yml", "down", "-v")
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	
	err := cmd.Run()
	if err != nil {
		log.Printf("Erreur suppression conteneurs: %v", err)
	} else {
		log.Println("Conteneurs et volumes supprimes")
	}
}

// cleanupTempFiles nettoie les fichiers temporaires
func cleanupTempFiles(verbose bool) {
	log.Println("=== NETTOYAGE DES FICHIERS TEMPORAIRES ===")

	tempPaths := []string{
		"./bin",           // Binaires compiles
		"./logs",          // Fichiers de logs (s'ils existent)
		"./tmp",           // Fichiers temporaires (s'ils existent)
	}

	for _, path := range tempPaths {
		if _, err := os.Stat(path); err == nil {
			if verbose {
				log.Printf("Suppression de %s...", path)
			}
			
			err := os.RemoveAll(path)
			if err != nil {
				log.Printf("Erreur suppression %s: %v", path, err)
			} else {
				log.Printf("Repertoire %s supprime", path)
			}
		} else if verbose {
			log.Printf("Repertoire %s n'existe pas", path)
		}
	}
}

// showPostCleanupInfo affiche les informations post-nettoyage
func showPostCleanupInfo(fullCleanup bool) {
	log.Println("")
	log.Println("Informations Post-Nettoyage")
	log.Println("===========================")

	if fullCleanup {
		log.Println("Nettoyage complet effectue:")
		log.Println("  - Bases de donnees e-commerce supprimees")
		log.Println("  - Services CouchDB arretes")
		log.Println("  - Conteneurs et volumes Docker supprimes")
		log.Println("  - Fichiers temporaires nettoyes")
		log.Println("")
		log.Println("L'environnement est pret pour une nouvelle installation:")
		log.Println("  1. make docker-up")
		log.Println("  2. make build")
		log.Println("  3. ./bin/setup")
		log.Println("  4. ./bin/loader -csv ./csv_files")
	} else {
		log.Println("Nettoyage partiel effectue")
		log.Println("Pour un nettoyage complet: ./bin/cleanup -full")
	}

	log.Println("")
	log.Println("Commandes utiles:")
	log.Println("  Verification Docker: docker ps")
	log.Println("  Verification volumes: docker volume ls")
	log.Println("  Redemarrage: make demo")
	log.Println("  Aide: ./bin/cleanup -h")
}