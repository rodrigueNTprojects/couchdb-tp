// cmd/setup/main.go
// Programme de configuration du cluster CouchDB e-commerce
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"couchdb-tp/pkg/cluster"
)

func main() {
	var (
		mode    = flag.String("mode", "setup", "Mode d'operation: setup, verify, ou cleanup")
		verbose = flag.Bool("verbose", false, "Affichage detaille")
	)
	flag.Parse()

	// Configuration du format des logs
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Execution selon le mode demande
	switch *mode {
	case "setup":
		setupCluster(*verbose)
	case "verify":
		verifyCluster(*verbose)
	case "cleanup":
		cleanupCluster(*verbose)
	default:
		log.Printf("Usage: %s [-mode setup|verify|cleanup] [-verbose]\n", os.Args[0])
		os.Exit(1)
	}
}

// setupCluster configure completement le cluster CouchDB
func setupCluster(verbose bool) {
	log.Println("Configuration du Cluster CouchDB E-commerce")
	log.Println("===========================================")

	// Verification des prerequis
	if !checkPrerequisites() {
		log.Println("Prerequis non satisfaits")
		os.Exit(1)
	}

	// Demarrage du cluster Docker
	if !startDockerCluster() {
		log.Println("Echec du demarrage Docker")
		os.Exit(1)
	}

	// Configuration du cluster
	config := cluster.NewClusterConfig()

	log.Println("Attente de la disponibilite des noeuds...")
	if err := config.WaitForNodes(); err != nil {
		log.Fatalf("Erreur attente noeuds: %v", err)
	}

	log.Println("Creation des bases de donnees systeme...")
	if err := config.CreateSystemDatabases(); err != nil {
		log.Fatalf("Erreur creation bases systeme: %v", err)
	}

	log.Println("Creation des bases de donnees e-commerce...")
	if err := config.CreateEcommerceDatabases(); err != nil {
		log.Fatalf("Erreur creation bases e-commerce: %v", err)
	}

	log.Println("Configuration de la replication...")
	if err := config.SetupReplication(); err != nil {
		log.Fatalf("Erreur configuration replication: %v", err)
	}

	log.Println("")
	log.Println("Configuration du cluster terminee avec succes!")
	
	showConnectionInfo(config)
}

// verifyCluster verifie l'etat actuel du cluster
func verifyCluster(verbose bool) {
	log.Println("Verification du Cluster CouchDB")
	log.Println("===============================")

	config := cluster.NewClusterConfig()
	if err := config.GetStatus(); err != nil {
		log.Fatalf("Erreur verification statut cluster: %v", err)
	}
	
	showConnectionInfo(config)
}

// cleanupCluster nettoie le cluster
func cleanupCluster(verbose bool) {
	log.Println("Nettoyage du Cluster CouchDB")
	log.Println("============================")

	log.Print("Etes-vous sur de vouloir supprimer le cluster? (oui/non): ")
	var response string
	fmt.Scanln(&response)

	if response == "oui" {
		log.Println("Suppression du cluster en cours...")
		cmd := exec.Command("docker-compose", "-f", "docker/docker-compose-cluster.yml", "down", "-v")
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Erreur suppression cluster: %v\nSortie: %s", err, output)
		} else {
			log.Println("Cluster supprime avec succes")
		}
	} else {
		log.Println("Nettoyage annule")
	}
}

// checkPrerequisites verifie que tous les prerequis sont presents
func checkPrerequisites() bool {
	log.Println("Verification des prerequis...")

	// Verification de Docker
	if _, err := exec.LookPath("docker"); err != nil {
		log.Println("Docker non trouve")
		return false
	}
	log.Println("Docker trouve")

	// Verification de docker-compose
	if _, err := exec.LookPath("docker-compose"); err != nil {
		log.Println("docker-compose non trouve")
		return false
	}
	log.Println("docker-compose trouve")

	// Verification du fichier docker-compose
	if _, err := os.Stat("docker/docker-compose-cluster.yml"); os.IsNotExist(err) {
		log.Println("Fichier docker-compose-cluster.yml non trouve")
		return false
	}
	log.Println("Fichier docker-compose-cluster.yml trouve")

	return true
}

// startDockerCluster demarre les conteneurs Docker
func startDockerCluster() bool {
	log.Println("Demarrage du cluster Docker...")

	cmd := exec.Command("docker-compose", "-f", "docker/docker-compose-cluster.yml", "up", "-d")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Erreur demarrage cluster Docker: %v\nSortie: %s", err, output)
		return false
	}

	log.Println("Conteneurs Docker demarres")
	return true
}

// showConnectionInfo affiche les informations de connexion
func showConnectionInfo(config *cluster.ClusterConfig) {
	log.Println("")
	log.Println("Informations de Connexion")
	log.Println("=========================")

	for _, node := range config.Nodes {
		log.Printf("%s:", node.Description)
		log.Printf("  URL: %s", node.URL)
		log.Printf("  Interface Fauxton: %s/_utils", node.URL)
		log.Println("")
	}

	log.Printf("Identifiants: %s / %s", config.Username, config.Password)
	log.Println("")
	log.Println("Prochaines etapes:")
	log.Println("  1. Verifier avec: ./bin/setup -mode verify (Linux/macOS) ou bin\\setup.exe -mode verify (Windows)")
	log.Println("  2. Surveiller avec: ./bin/monitor -database ecommerce_orders (Linux/macOS) ou bin\\monitor.exe -database ecommerce_orders (Windows)")
	
}