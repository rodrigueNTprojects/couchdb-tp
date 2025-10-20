// cmd/scenario_offline_equipB.go
// Scénario Équipe B : Panne pendant réplication (Nœud EU1)
// Programme Go cross-platform pour tester la résilience

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"

	"couchdb-tp/pkg/client"
)

func main() {
	log.Println("=== SCENARIO EQUIPE B : PANNE EU1 PENDANT REPLICATION ===")

	// 1. Créer des clients pour NA1 et NA2
	na1Client, err := client.New("http://localhost:5987", "admin", "ecommerce2024")
	if err != nil {
		log.Fatalf("Erreur création client NA1: %v", err)
	}

	na2Client, err := client.New("http://localhost:5988", "admin", "ecommerce2024")
	if err != nil {
		log.Fatalf("Erreur création client NA2: %v", err)
	}

	log.Println("Étape 0/6 : Nettoyage des documents existants...")
	
	// Supprimer le document s'il existe déjà
	deleteExistingDoc(na1Client, "/ecommerce_leads/lead_sync_conflict")
	
	// Attendre un peu pour la réplication de la suppression
	time.Sleep(3 * time.Second)

	log.Println("Étape 1/6 : Création du document initial sur NA1...")

	// Créer un document initial sur NA1
	initialDoc := map[string]interface{}{
		"_id":            "lead_sync_conflict",
		"ledger_type":    "sales_pipeline",
		"pipeline_stage": "qualified",
		"lead_hash":      "initial_hash",
		"audit_trail": map[string]interface{}{
			"created_by":      "scenario_b",
			"created_at":      time.Now().Format(time.RFC3339),
			"source_node":     "NA1",
			"validation_hash": "initial_hash",
			"integrity_check": "validated",
			"ledger_version":  "1.0",
		},
		"lead_data": map[string]interface{}{
			"mql_id": "sync_test",
			"qualification": map[string]interface{}{
				"first_contact_date": "2025-10-18",
				"origin":             "test_scenario",
				"landing_page_id":    "test_page",
			},
		},
	}

	resp, err := na1Client.Put("/ecommerce_leads/lead_sync_conflict", initialDoc)
	if err != nil {
		log.Fatalf("Erreur création document initial: %v", err)
	}

	// Récupérer le _rev du document créé avec vérifications
	var createResp map[string]interface{}
	if err := json.Unmarshal(resp.Body, &createResp); err != nil {
		log.Fatalf("Erreur parsing réponse création: %v\nRéponse brute: %s", err, string(resp.Body))
	}

	// Vérifier s'il y a une erreur dans la réponse
	if errMsg, hasError := createResp["error"]; hasError {
		log.Fatalf("Erreur CouchDB lors de la création: %v - %v", errMsg, createResp["reason"])
	}

	// Vérifier que 'rev' existe et est bien une string
	var originalRev string
	if revInterface, exists := createResp["rev"]; exists && revInterface != nil {
		if revStr, ok := revInterface.(string); ok {
			originalRev = revStr
		} else {
			log.Fatalf("Le champ 'rev' n'est pas une string: %v", revInterface)
		}
	} else {
		log.Fatalf("Le champ 'rev' n'existe pas dans la réponse: %+v", createResp)
	}

	log.Printf("✓ Document initial créé - Status: %d, Rev: %s", resp.StatusCode, originalRev)

	// 2. Attendre la réplication
	log.Println("Étape 2/6 : Attente de la réplication (5 secondes)...")
	time.Sleep(5 * time.Second)

	// 3. Arrêter EU1
	log.Println("Étape 3/6 : SIMULATION - Arrêt du nœud EU1...")
	if err := dockerCommand("stop", "couchdb-eu-1"); err != nil {
		log.Printf("Avertissement arrêt EU1: %v (conteneur peut-être déjà arrêté)", err)
	}
	time.Sleep(3 * time.Second)
	log.Println("✓ EU1 arrêté")

	// 4. Modifier sur NA1
	log.Println("Étape 4/6 : Modification sur NA1 (version A)...")

	// Récupérer le document actuel pour obtenir le _rev à jour
	getResp, err := na1Client.Get("/ecommerce_leads/lead_sync_conflict")
	if err != nil {
		log.Fatalf("Erreur récupération document: %v", err)
	}

	var currentDoc map[string]interface{}
	if err := json.Unmarshal(getResp.Body, &currentDoc); err != nil {
		log.Fatalf("Erreur parsing document actuel: %v", err)
	}

	// Vérification sécurisée du _rev
	currentRev := extractRev(currentDoc, "_rev")
	log.Printf("Révision actuelle: %s", currentRev)

	// Modifier sur NA1
	docNA1 := map[string]interface{}{
		"_id":            "lead_sync_conflict",
		"_rev":           currentRev,
		"ledger_type":    "sales_pipeline",
		"pipeline_stage": "closed",
		"lead_hash":      "hash_na1",
		"audit_trail": map[string]interface{}{
			"created_by":      "scenario_b",
			"created_at":      time.Now().Format(time.RFC3339),
			"source_node":     "NA1",
			"validation_hash": "hash_na1",
			"integrity_check": "validated",
			"ledger_version":  "1.0",
		},
		"lead_data": map[string]interface{}{
			"mql_id": "sync_test",
			"qualification": map[string]interface{}{
				"first_contact_date": "2025-10-18",
				"origin":             "test_scenario",
				"landing_page_id":    "test_page",
			},
			"conversion": map[string]interface{}{
				"won_date":         time.Now().Format("2006-01-02"),
				"seller_id":        "seller_na1",
				"business_segment": "test_na1",
			},
		},
	}

	resp, err = na1Client.Put("/ecommerce_leads/lead_sync_conflict", docNA1)
	if err != nil {
		log.Printf("Erreur modification NA1: %v", err)
	} else {
		log.Printf("✓ Modifié sur NA1 - Status: %d", resp.StatusCode)
	}

	// 5. Modifier sur NA2 avec la révision ORIGINALE (crée le conflit)
	log.Println("Étape 5/6 : Modification sur NA2 avec révision originale (version B - CONFLIT)...")

	docNA2 := map[string]interface{}{
		"_id":            "lead_sync_conflict",
		"_rev":           originalRev, // UTILISE LA RÉVISION ORIGINALE = CONFLIT
		"ledger_type":    "sales_pipeline",
		"pipeline_stage": "qualified",
		"lead_hash":      "hash_na2",
		"audit_trail": map[string]interface{}{
			"created_by":      "scenario_b",
			"created_at":      time.Now().Add(10 * time.Second).Format(time.RFC3339),
			"source_node":     "NA2",
			"validation_hash": "hash_na2",
			"integrity_check": "validated",
			"ledger_version":  "1.0",
		},
		"lead_data": map[string]interface{}{
			"mql_id": "sync_test",
			"qualification": map[string]interface{}{
				"first_contact_date": "2025-10-18",
				"origin":             "test_scenario_na2",
				"landing_page_id":    "test_page_na2",
			},
			"conversion": map[string]interface{}{
				"won_date":         time.Now().Format("2006-01-02"),
				"seller_id":        "seller_na2",
				"business_segment": "test_na2",
			},
		},
	}

	resp, err = na2Client.Put("/ecommerce_leads/lead_sync_conflict", docNA2)
	if err != nil {
		log.Printf("Modification NA2: %v", err)
	} else {
		log.Printf("✓ Modification NA2 avec révision originale - Status: %d (CONFLIT CRÉÉ)", resp.StatusCode)
	}

	// 6. Redémarrer EU1 et attendre synchronisation
	log.Println("Étape 6/6 : Redémarrage EU1 et synchronisation...")
	if err := dockerCommand("start", "couchdb-eu-1"); err != nil {
		log.Printf("Avertissement redémarrage EU1: %v", err)
	}

	// Attendre la synchronisation
	log.Println("Attente de la synchronisation complète...")
	time.Sleep(15 * time.Second)

	// Vérifier les conflits sur différents nœuds
	log.Println("\nAnalyse des délais de synchronisation...")

	// Test sur NA1
	resp, err = na1Client.Get("/ecommerce_leads/lead_sync_conflict?conflicts=true")
	if err != nil {
		log.Printf("Erreur vérification NA1: %v", err)
	} else {
		log.Printf("NA1 synchronisé - Status: %d", resp.StatusCode)
		
		var na1Doc map[string]interface{}
		json.Unmarshal(resp.Body, &na1Doc)
		
		if conflicts, ok := na1Doc["_conflicts"]; ok {
			conflictList := conflicts.([]interface{})
			log.Printf("\n✓✓✓ CONFLIT DÉTECTÉ SUR NA1: %d révision(s) conflictuelle(s) ✓✓✓", len(conflictList))
			log.Printf("Révisions en conflit: %v", conflicts)
			fmt.Printf("\nRéponse NA1: %s\n", string(resp.Body))
		} else {
			log.Println("\n✗ Aucun conflit détecté sur NA1")
		}
	}

	// Test sur NA2
	resp, err = na2Client.Get("/ecommerce_leads/lead_sync_conflict?conflicts=true")
	if err != nil {
		log.Printf("Erreur vérification NA2: %v", err)
	} else {
		log.Printf("NA2 synchronisé - Status: %d", resp.StatusCode)
		
		var na2Doc map[string]interface{}
		json.Unmarshal(resp.Body, &na2Doc)
		
		if conflicts, ok := na2Doc["_conflicts"]; ok {
			conflictList := conflicts.([]interface{})
			log.Printf("\n✓✓✓ CONFLIT DÉTECTÉ SUR NA2: %d révision(s) conflictuelle(s) ✓✓✓", len(conflictList))
		} else {
			log.Println("\n✗ Aucun conflit détecté sur NA2")
		}
	}

	// Test sur EU1 (après redémarrage)
	eu1Client, err := client.New("http://localhost:5989", "admin", "ecommerce2024")
	if err == nil {
		resp, err := eu1Client.Get("/ecommerce_leads/lead_sync_conflict?conflicts=true")
		if err != nil {
			log.Printf("Erreur vérification EU1: %v", err)
		} else {
			log.Printf("EU1 synchronisé - Status: %d", resp.StatusCode)
			
			var eu1Doc map[string]interface{}
			json.Unmarshal(resp.Body, &eu1Doc)
			
			if conflicts, ok := eu1Doc["_conflicts"]; ok {
				conflictList := conflicts.([]interface{})
				log.Printf("\n✓✓✓ CONFLIT DÉTECTÉ SUR EU1: %d révision(s) conflictuelle(s) ✓✓✓", len(conflictList))
			} else {
				log.Println("\n✗ Aucun conflit détecté sur EU1")
			}
		}
	}

	log.Println("\n=== SCENARIO EQUIPE B TERMINE ===")
	log.Println("")
	log.Println("Instructions de test des vues:")
	log.Println("  1. Testez votre vue 'pipeline_resilience' pour voir l'impact sur les leads")
	log.Println("  2. Vérifiez 'post_incident_distribution' pour analyser la redistribution")
	log.Println("  3. Utilisez le resolver pour résoudre le conflit généré:")
	log.Println("     Windows: .\\bin\\conflict-resolver.exe -database ecommerce_leads -list")
	log.Println("     Linux/macOS: ./bin/conflict-resolver -database ecommerce_leads -list")
}

// deleteExistingDoc supprime un document s'il existe
func deleteExistingDoc(client *client.Client, path string) {
	// Essayer de récupérer le document
	resp, err := client.Get(path)
	if err != nil {
		log.Printf("Document n'existe pas ou erreur: %v (c'est normal au premier lancement)", err)
		return
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(resp.Body, &doc); err != nil {
		log.Printf("Erreur parsing document existant: %v", err)
		return
	}

	// Récupérer le _rev
	if revInterface, exists := doc["_rev"]; exists && revInterface != nil {
		if rev, ok := revInterface.(string); ok {
			// Supprimer le document
			deleteResp, err := client.Delete(path + "?rev=" + rev)
			if err != nil {
				log.Printf("Erreur suppression document: %v", err)
			} else {
				log.Printf("✓ Document existant supprimé (rev: %s, status: %d)", rev, deleteResp.StatusCode)
			}
		}
	}
}

// extractRev extrait de manière sécurisée le _rev d'un document
func extractRev(doc map[string]interface{}, fieldName string) string {
	if revInterface, exists := doc[fieldName]; exists && revInterface != nil {
		if revStr, ok := revInterface.(string); ok {
			return revStr
		}
		log.Fatalf("Le champ '%s' n'est pas une string: %v", fieldName, revInterface)
	}
	log.Fatalf("Le champ '%s' n'existe pas dans le document: %+v", fieldName, doc)
	return ""
}

// dockerCommand exécute une commande Docker de manière cross-platform
func dockerCommand(action, container string) error {
	cmd := exec.Command("docker", action, container)
	return cmd.Run()
}