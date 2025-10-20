# TP CouchDB - Registres Distribués pour E-commerce (IND500)

## README - Guide de démarrage

### **Prérequis**

- Go 1.19+ installé ([télécharger](https://go.dev/dl/))
- Docker Desktop
  ([télécharger](https://www.docker.com/products/docker-desktop))
- Git ([télécharger](https://git-scm.com/downloads))
- Navigateur web moderne

### **Installation**

    # Cloner le projet
    git clone https://github.com/rodrigueNTprojects/couchdb-tp.git
    cd couchdb-tp

    # Télécharger les dépendances Go
    go mod download

### **Compilation des outils**

**Windows (PowerShell/CMD) :**

    go build -o bin\setup.exe .\cmd\setup
    go build -o bin\loader.exe .\cmd\loader
    go build -o bin\monitor.exe .\cmd\monitor
    go build -o bin\cleanup.exe .\cmd\cleanup

**Linux/macOS (Terminal) :**

    go build -o bin/setup ./cmd/setup
    go build -o bin/loader ./cmd/loader
    go build -o bin/monitor ./cmd/monitor
    go build -o bin/cleanup ./cmd/cleanup

**Alternative avec Makefile (Linux/macOS uniquement) :**

    make build

### **Démarrage du cluster Docker**

**Tous systèmes :**

    docker-compose -f docker/docker-compose-cluster.yml up -d

    # Vérifier que les 4 conteneurs sont démarrés
    docker ps

### **Test de l\'installation**

**Accéder aux interfaces Fauxton :**

- **NA1** (Principal) : <http://localhost:5987/_utils>
- **NA2** : <http://localhost:5988/_utils>
- **EU1** : <http://localhost:5989/_utils>
- **AP1** : <http://localhost:5990/_utils>

**Identifiants** : `admin` / `ecommerce2024`

Si les 4 interfaces s\'affichent correctement, votre installation est
prête !

## Contexte du projet

Un système de commerce électronique nécessite un registre distribué pour
tracer de manière immutable toutes les transactions commerciales,
définitions produites, partenaires et prospects. Contrairement à une
base de données traditionnelle, ce registre doit garantir l\'intégrité,
la traçabilité et la non-répudiation des données sur plusieurs sites
géographiques.

**Architecture cible :**

- 4 nœuds CouchDB simulant des régions (Amérique du Nord, Europe,
  Asie-Pacifique)
- Réplication bidirectionnelle pour la synchronisation des registres
- Documents JSON avec audit trail complet et vérification d\'intégrité
- Gestion des conflits et résolution automatique

## Données et registres

Le système gère 4 types de registres distribués basés sur des tables SQL
existantes :

### **Registres à implémenter**

- **ecommerce_orders** : Registre des transactions commerciales
  immutables
- **ecommerce_products** : Registre de définitions produits avec
  versioning
- **ecommerce_sellers** : Registre des partenaires certifiés
- **ecommerce_leads** : Registre du pipeline de vente avec traçabilité

### Tables SQL sources

Le TP se base sur un modèle e-commerce SQL : `customers`, `geolocation`,
`orders`, `order_items`, `order_payments`, `order_reviews`, `products`,
`product_category_name_translation`, `sellers`, `leads_qualified`,
`leads_closed`

## Structure du projet

    couchdb-tp/
    ├── README.md                          # Ce fichier
    ├── go.mod                             # Dépendances Go
    ├── Makefile                           # Automatisation (Linux/macOS)
    ├── docker/
    │   └── docker-compose-cluster.yml     # Configuration 4 nœuds
    ├── data/                              # Fichiers CSV source
    ├── cmd/                               # Applications exécutables
    │   ├── setup/main.go                  # Configuration cluster
    │   ├── loader/main.go                 # Chargement données
    │   ├── monitor/main.go                # Surveillance
    │   ├── cleanup/main.go                # Nettoyage
    │   ├── scenario_offline_equipA.go     # Scénario panne équipe A
    │   ├── scenario_offline_equipB.go     # Scénario panne équipe B
    │   └── conflict-resolver/main.go      # Résolution conflits
    ├── pkg/                               # Packages réutilisables
    │   ├── client/client.go               # Client HTTP CouchDB
    │   ├── cluster/cluster.go             # Gestion cluster
    │   ├── loader/loader.go               # Logique chargement
    │   ├── monitor/monitor.go             # Logique surveillance
    │   └── conflict/resolver.go           # Résolution conflits
    ├── bin/                               # Binaires compilés
    └── config.go                          # Configuration cluster (à compléter)

## Étape 0 : Configuration Cluster Distribué (25 points)

### **Objectifs**

- Comprendre l\'architecture de réplication bidirectionnelle
- Compléter la configuration d\'un cluster 4 nœuds
- Valider la synchronisation entre registres

### **Travail à réaliser**

#### **Mission 1 : Compléter la configuration de réplication**

Le fichier `config.go` configure actuellement la réplication entre 2
nœuds (NA1 ↔ NA2).

**Votre tâche :** Ajouter 8 réplications bidirectionnelles manquantes :

- NA1 ↔ EU1 (2 réplications)
- NA1 ↔ AP1 (2 réplications)
- NA2 ↔ EU1 (2 réplications)
- NA2 ↔ AP1 (2 réplications)
- EU1 ↔ AP1 (2 réplications)

**Exemple de format :**

    {Source: "NA1", Target: "EU1", Name: "na1_to_eu1"},
    {Source: "EU1", Target: "NA1", Name: "eu1_to_na1"},

#### **Mission 2 : Compiler et exécuter la configuration**

**Windows :**

    # Recompiler après modification
    go build -o bin\setup.exe .\cmd\setup

    # Exécuter la configuration
    .\bin\setup.exe -mode setup

**Linux/macOS :**

    # Recompiler après modification
    go build -o bin/setup ./cmd/setup

    # Exécuter la configuration
    ./bin/setup -mode setup

#### Mission 3 : Vérifier la réplication

**Tous systèmes :**

    # Vérifier le statut du cluster
    ./bin/setup -mode verify        # Linux/macOS
    .\bin\setup.exe -mode verify    # Windows

    # Surveiller une base
    ./bin/monitor -database ecommerce_orders        # Linux/macOS
    .\bin\monitor.exe -database ecommerce_orders    # Windows

### **Livrables Étape 0**

- **Code complété** : Fichier `config.go` avec les 8 réplications
  manquantes
- **Capture d\'écran** : Les 4 interfaces Fauxton actives montrant les
  bases de données créées
- **Résultat de vérification** : Sortie de la commande de vérification
- **Schéma d\'architecture** : Diagramme comprenant :
  - 4 nœuds (NA1, NA2, EU1, AP1) avec régions géographiques
  - 12 réplications bidirectionnelles avec flèches et noms
  - 4 bases de données répliquées sur chaque nœud
  - Légende des types de réplication

## Étape 1 : Chargement et Structuration des Registres (30 points)

### **Objectifs**

- Charger les données CSV dans le cluster distribué
- Comprendre la transformation SQL → NoSQL avec audit trail
- Valider la réplication entre tous les nœuds

### **Schémas de transformation SQL → NoSQL**

#### 1. ecommerce_orders (Registre des transactions)

**Sources SQL** : `orders` + `order_items` + `order_payments` +
`order_reviews` + `customers` + `geolocation`

    {
      "_id": "ledger_transaction_[order_id]",
      "ledger_type": "commercial_transaction",
      "immutable": true,
      "transaction_hash": "sha256_hash",
      "timestamp": "order_purchase_timestamp",
      "audit_trail": {
        "created_by": "csv_loader_system",
        "created_at": "2024-01-15T10:30:00Z",
        "source_node": "NA1",
        "validation_hash": "sha256_hash",
        "integrity_check": "validated",
        "ledger_version": "1.0"
      },
      "transaction_data": {
        "order_id": "...",
        "customer": {
          "customer_id": "...",
          "location": {
            "city": "customer_city",
            "state": "customer_state",
            "coordinates": [lat, lng]
          }
        },
        "order_details": {
          "items": [...],
          "totals": {...}
        },
        "payments": [...],
        "reviews": [...]
      }
    }

#### 2. ecommerce_products (Registre du catalogue)

**Sources SQL** : `products` + `product_category_name_translation`

    {
      "_id": "ledger_product_[product_id]_v1",
      "ledger_type": "product_definition", 
      "version": 1,
      "product_hash": "sha256_hash",
      "audit_trail": {...},
      "product_data": {
        "product_id": "...",
        "category": "product_category_name",
        "category_english": "...",
        "specifications": {
          "dimensions": {
            "weight_g": "...",
            "length_cm": "...",
            "height_cm": "...",
            "width_cm": "..."
          }
        }
      }
    }

#### 3. ecommerce_sellers (Registre des partenaires)

**Sources SQL** : `sellers` + `geolocation`

    {
      "_id": "ledger_seller_[seller_id]",
      "ledger_type": "partner_registry",
      "certification_status": "active",
      "seller_hash": "sha256_hash",
      "audit_trail": {...},
      "seller_data": {
        "seller_id": "...",
        "business_info": {
          "city": "seller_city",
          "state": "seller_state"
        },
        "location": {
          "coordinates": [lat, lng],
          "region": "north_america|europe|asia_pacific"
        }
      }
    }

#### 4. ecommerce_leads (Registre du pipeline)

**Sources SQL** : `leads_qualified` + `leads_closed`

    {
      "_id": "ledger_lead_[mql_id]",
      "ledger_type": "sales_pipeline",
      "pipeline_stage": "qualified|closed",
      "lead_hash": "sha256_hash",
      "audit_trail": {...},
      "lead_data": {
        "mql_id": "...",
        "qualification": {...},
        "conversion": {...}
      }
    }

### **Éléments d\'audit ajoutés (nouveaux par rapport au SQL)**

- `audit_``trail` : Traçabilité complète
- `*``_hash` : Hash SHA-256 pour intégrité
- `ledger``_type` : Classification du registre
- `immutable` : Marqueur de non-modification
- `version` : Versioning des produits
- `source_``node` : Nœud de création

### **Travail à réaliser**

#### **Mission 1 : Analyser la transformation**

Examinez `pkg``/loader/``loader.go` qui transforme automatiquement les
CSV en registres enrichis.

#### **Mission 2 : Charger les données**

**Windows :**

    .\bin\loader.exe -csv .\data -verbose

**Linux/macOS :**

    ./bin/loader -csv ./data -verbose

#### **Mission 3 : Valider la réplication**

**Tous systèmes :**

    # Surveiller chaque base
    ./bin/monitor -database ecommerce_orders     # Linux/macOS
    ./bin/monitor -database ecommerce_products
    ./bin/monitor -database ecommerce_sellers
    ./bin/monitor -database ecommerce_leads

    # Windows : remplacer ./bin/monitor par .\bin\monitor.exe

### **Livrables Étape 1**

- **Statistiques** : Nombre de documents par registre par nœud
- **Exemples JSON** : 1 document par registre avec audit trail visible
- **Schémas de transformation** : Mapping détaillé SQL → NoSQL pour
  chaque registre
- **Preuve de cohérence** : Les 4 nœuds contiennent les mêmes données

## Étape 2 : Vues d\'Audit et Traçabilité avec Scénarios Offline (35 points)

### **Objectifs**

- Créer des vues Map-Reduce dans **Fauxton**
- Simuler des scénarios offline
- Analyser la traçabilité et détecter les conflits
- Résoudre les conflits automatiquement

### **Phase 2A : Scénarios Offline par Équipe (15 points)**

#### **Compilation des scénarios**

**Windows :**

    go build -o bin\scenario-a.exe .\cmd\scenario_offline_equipA.go
    go build -o bin\scenario-b.exe .\cmd\scenario_offline_equipB.go

**Linux/macOS :**

    go build -o bin/scenario-a ./cmd/scenario_offline_equipA.go
    go build -o bin/scenario-b ./cmd/scenario_offline_equipB.go

#### **Exécution des scénarios**

**Équipe A :**

    ./bin/scenario-a        # Linux/macOS
    .\bin\scenario-a.exe    # Windows

**Équipe B :**

    ./bin/scenario-b        # Linux/macOS
    .\bin\scenario-b.exe    # Windows

Ces programmes simulent automatiquement : création de documents de test,
arrêt de nœuds, modifications concurrentes, redémarrage et détection de
conflits.

#### **Résolution de conflits**

**Flow de résolution :**

    Conflit détecté → Récupérer révisions → Comparer timestamps → Garder la plus récente → Supprimer l'ancienne

**Fichier à compléter** : `pkg``/``conflict``/``resolver.go` (3-4 lignes
d'instruction). Le fichier resolver_instructions peut être utilisé pour
faciliter ce travail.

Après avoir completer le fichier resolver.go, il faut le compiler,
lister les conflits qui ont été créés par les scénarios pour chaque
équipe et, corriger les conflits

**Compilation du résolveur :**

**Windows :**

    go build -o bin\conflict-resolver.exe .\cmd\conflict-resolver

**Linux/macOS :**

    go build -o bin/conflict-resolver ./cmd/conflict-resolver

**Utilisation :**

    # Lister les conflits
    # Linux/macOS
    ./bin/conflict-resolver -database ecommerce_orders -list
    ./bin/conflict-resolver -database ecommerce_leads -list
    # Windows
    .\bin\conflict-resolver.exe -database ecommerce_orders -list
    .\bin\conflict-resolver.exe -database ecommerce_leads -list

    # Résoudre un conflit
    # Linux/macOS
    ./bin/conflict-resolver -database ecommerce_orders -doc test_order_conflict
    ./bin/conflict-resolver -database ecommerce_leads -doc lead_sync_conflict
    # Linux/macOS
    .\bin\conflict-resolver.exe -database ecommerce_orders -doc test_order_conflict
    .\bin\conflict-resolver.exe -database ecommerce_leads -doc lead_sync_conflict

### **Phase 2B : Vues d\'Audit et Traçabilité (20 points)**

**Important** : Créer les vues dans **Fauxton** (interface web CouchDB).

### **Équipe A : 5 vues d\'intégrité et audit**

#### Vue 1A : Vérification d\'intégrité des transactions

**Base** : `ecommerce_orders` \| **Design** : `_design/audit` \| **Vue**
: `integrity_verification`` `\| **Reduce** : `Aucune`

    function(doc) {
      if (doc.ledger_type === 'commercial_transaction' && doc.transaction_hash) {
        emit(doc.audit_trail.source_node, {
          order_id: doc.transaction_data ? doc.transaction_data.order_id : null,
          transaction_hash: doc.transaction_hash,
          integrity_status: doc.audit_trail.integrity_check,
          created_at: doc.audit_trail.created_at
        });
      }
    }

#### Vue 2A : Audit trail avec marqueurs de panne

**Base** : `ecommerce_orders` \| **Design** : `_design/audit` \| **Vue**
: `outage_impact_audit`` `\| **Reduce** : `Aucune`

    function(doc) {
      if (doc.audit_trail && doc.audit_trail.created_at) {
        var date = new Date(doc.audit_trail.created_at);
        var month = date.getFullYear() + '-' + (date.getMonth() + 1).toString().padStart(2, '0');
        var duringOutage = doc.audit_trail.modification === 'during_na2_outage' || 
                          (doc.transaction_data && doc.transaction_data.modified_during_outage);
        
        emit([month, doc.ledger_type, duringOutage ? 'during_outage' : 'normal'], {
          id: doc._id,
          source_node: doc.audit_trail.source_node,
          outage_marker: duringOutage
        });
      }
    }

#### Vue 3A : Distribution géographique (À COMPLÉTER)

**Base** : `ecommerce_orders` \| **Design** : `_design/audit` \| **Vue**
: `geographic_distribution`` `\| **Reduce** : `Aucune`

**Logique à implémenter :**

1.  Vérifier `doc.audit``_trail.source_node`
2.  Extraire région : `doc.audit``_trail.source_node.substring``(0, 2)`
    (NA, EU, AP)
3.  Émettre avec clé `[``region``, ``ledger_type``]`
4.  Valeur :
    `{``doc_count``: 1, ledger_size: JSON.stringify(doc).length, integrity_hash: doc.audit_trail.validation_hash}`

#### Vue 4A : Synchronisation inter-nœuds

**Base** : `ecommerce_orders` \| **Design** : `_design/audit` \| **Vue**
: `node_sync_performance`` `\| **Reduce** : `Aucune`

    function(doc) {
      if (doc.audit_trail && doc._rev) {
        var revisionNumber = parseInt(doc._rev.split('-')[0]);
        emit([doc.audit_trail.source_node, doc.ledger_type], {
          doc_id: doc._id,
          revision_count: revisionNumber,
          has_conflicts: doc._conflicts ? doc._conflicts.length : 0,
          sync_indicator: revisionNumber > 1 ? 'replicated' : 'original'
        });
      }
    }

#### Vue 5A : Détection de conflits (requête lente)

**Base** : `ecommerce_orders` \| **Design** : `_design/audit` \| **Vue**
: `conflict_detection`` `\| **Reduce** : `_count`

    function(doc) {
      if (doc._conflicts) {
        emit(['conflict_detected', doc.ledger_type], {
          doc_id: doc._id,
          conflicts: doc._conflicts,
          audit_source: doc.audit_trail.source_node
        });
      }
      if (doc.audit_trail && doc.audit_trail.validation_hash) {
        emit(['integrity_check', doc.audit_trail.validation_hash], {
          doc_id: doc._id,
          ledger_type: doc.ledger_type
        });
      }
    }

### **Équipe B : 5 vues de traçabilité business**

#### Vue 1B : Pipeline de vente avec interruptions

**Base** : `ecommerce_leads` \| **Design** : `_design/sales_audit` \|
**Vue** : `pipeline_resilience`` `\| **Reduce** : `Aucune`

    function(doc) {
      if (doc.ledger_type === 'sales_pipeline') {
        var hasConflict = doc._conflicts && doc._conflicts.length > 0;
        var isTestLead = doc.lead_data.mql_id === 'sync_test' || doc._id.includes('lead_sync_conflict');
        
        emit([doc.pipeline_stage, hasConflict ? 'with_conflict' : 'clean'], {
          lead_id: doc.lead_data.mql_id,
          source_node: doc.audit_trail ? doc.audit_trail.source_node : 'unknown',
          test_scenario: isTestLead
        });
      }
    }

#### Vue 2B : Certification des partenaires (À COMPLÉTER)

**Base** : `ecommerce_sellers` \| **Design** : `_design/``sales_audit`
\| **Vue** : `partner_certification`` `\| **Reduce** : `Aucune`

**Logique à implémenter :**

1.  Vérifier `doc.ledger``_type`` === '``partner_registry``'`
2.  Émettre avec clé
    `[``doc.certification``_status``, ``doc.seller_data.location.region``]`
3.  Valeur :
    `{``seller_id``, ``business_city``, certification_date, certification_node, integrity_hash``}`

#### Vue 3B : Traçabilité des modifications

**Base** : `ecommerce_leads` \| **Design** : `_design/``sales_audit` \|
**Vue** : `user_action_audit`` `\| **Reduce** : `Aucune`

    function(doc) {
      if (doc.audit_trail) {
        emit([doc.audit_trail.created_by, doc.ledger_type], {
          action_timestamp: doc.audit_trail.created_at,
          document_id: doc._id,
          source_node: doc.audit_trail.source_node
        });
      }
    }

#### Vue 4B : Distribution post-incident

**Base** : `ecommerce_leads` \| **Design** : `_design/sales_audit` \|
**Vue** : `post_incident_distribution`` `\| **Reduce** : `Aucune`

    function(doc) {
      if (doc.audit_trail && doc.audit_trail.source_node) {
        var nodeRegion = doc.audit_trail.source_node.substring(0, 2);
        var docStatus = doc.audit_trail.conflict_resolved_at ? 'resolved' : 
                       (doc._conflicts ? 'conflicted' : 'normal');
        
        emit([nodeRegion, doc.ledger_type, docStatus], {
          doc_count: 1,
          incident_impact: doc.audit_trail.conflict_resolved_at || doc._conflicts ? 'affected' : 'unaffected'
        });
      }
    }

#### Vue 5B : Analyse de synchronisation (requête lente)

**Base** : `ecommerce_leads` \| **Design** : `_design/sales_audit` \|
**Vue** : `synchronization_performance`` `\| **Reduce** : `CUSTOM`

    // Map function avec pré-calcul
    function(doc) {
      if (!doc.audit_trail || !doc._rev) {
        return;
      }
      
      var revisionNumber = parseInt(doc._rev.split('-')[0]);
      
      // Émettre seulement les informations essentielles
      if (revisionNumber > 1) {
        emit([doc.audit_trail.source_node, 'R'], 1); // R = Replicated
      } else {
        emit([doc.audit_trail.source_node, 'O'], 1); // O = Original
      }
    }

    // Reduce optimisé : simple comptage
    function(keys, values, rereduce) {
      return sum(values);
    }

### 

### **Tests des vues avec curl**

Tous systèmes (curl disponible nativement) :

    # Équipe A - Tests
    curl http://admin:ecommerce2024@localhost:5987/ecommerce_orders/_design/audit/_view/integrity_verification?group=true
    curl http://admin:ecommerce2024@localhost:5987/ecommerce_orders/_design/audit/_view/outage_impact_audit?group=true
    curl http://admin:ecommerce2024@localhost:5987/ecommerce_orders/_design/audit/_view/geographic_distribution?group=true
    curl http://admin:ecommerce2024@localhost:5987/ecommerce_orders/_design/audit/_view/node_sync_performance?group=true
    curl http://admin:ecommerce2024@localhost:5987/ecommerce_orders/_design/audit/_view/conflict_detection?group=true

    # Équipe B - Tests
    curl http://admin:ecommerce2024@localhost:5987/ecommerce_leads/_design/sales_audit/_view/pipeline_resilience?group=true
    curl http://admin:ecommerce2024@localhost:5989/ecommerce_sellers/_design/sales_audit/_view/partner_certification?group=true
    curl http://admin:ecommerce2024@localhost:5987/ecommerce_leads/_design/sales_audit/_view/user_action_audit?group=true
    curl http://admin:ecommerce2024@localhost:5987/ecommerce_leads/_design/sales_audit/_view/post_incident_distribution?group=true
    curl http://admin:ecommerce2024@localhost:5987/ecommerce_leads/_design/sales_audit/_view/synchronization_performance?reduce=true&group=true 

### **Livrables Étape 2**

- **Scripts de scénarios** : Logs de votre scénario d\'équipe
- **Code de résolution** : `pkg``/``conflict``/``resolver.go` complété
- **Design documents** : Captures Fauxton des vues créées
- **Tests curl** : Résultats avant/après scénarios
- **Analyse** : explication de chaque requête et du resultat de son test

## Étape 3 : Optimisation pour Registres Distribués (20 points)

### **Objectifs**

- Optimiser la vue 5 de votre équipe
- Mesurer l\'impact sur les performances
- Proposer des améliorations

### **1. Analyse des performances**

**Tous systèmes (mesure avec curl) :**

**Équipe A :**

\# 1. Mesurer le temps de réponse

curl -w \"\\nTemps: %{time_total}s\\n\"
\"http://admin:ecommerce2024@localhost:5987/ecommerce_orders/\_design/audit/\_view/conflict_detection?group=true\"

*\# 2. Noter le nombre de documents traités*

curl \"http://admin:ecommerce2024@localhost:5987/ecommerce_orders\" \|
findstr \"doc_count\"

**Équipe B :**

    curl -w "\nTemps NA1: %{time_total}s\n" http://admin:ecommerce2024@localhost:5987/ecommerce_leads/_design/sales_audit/_view/synchronization_performance?reduce=true&group=true
    curl "http://admin:ecommerce2024@localhost:5987/ecommerce_leads" | findstr "doc_count"

### **2. Implémentation d\'une optimisation**

**Option équipe A : Base** : `ecommerce_orders` \| **Design** :
`_design/audit` \| **Vue** : `conflicts_only_fast`` `\| **Reduce** :
`Aucune`

Utiliser une fonction qui effectue un filtrage très tôt -- elle va se
focaliser que sur les documents avec conflits

Ou utiliser des index

**Option équipe B : Base** : `ecommerce_leads` \| **Design** :
`_design/``sales_audit` \| **Vue** :
`synchronization_performance``_optimized`` `\| **Reduce** : `count`

Cette vue va modifier la vue 5 en créant un index qui va permettre de
regrouper et interroger les documents en fonction du nœud source
(audit_trail) et du statut (la version original ou d'une version mise à
jour/repliquée). Elle va compter le nombre de documents qui
correspondent à la paire \[nœud_source, statut\]

### **3. Tests de performance comparative**

**Après optimisation :**

Équipe A :\
\# Vue originale sur NA1

curl -w \"\\nVue originale NA1: %{time_total}s\\n\"
<http://admin:ecommerce2024@localhost:5987/ecommerce_orders/_design/audit/_view/conflict_detection?group=true>

\# Vue optimisée sur NA1

curl -w \"\\nVue optimisée NA1: %{time_total}s\\n\"
<http://admin:ecommerce2024@localhost:5987/ecommerce_orders/_design/audit/_view/conflicts_only_fast>

\# Vue optimisée sur NA2

curl -w \"\\nVue optimisée NA2: %{time_total}s\\n\"
<http://admin:ecommerce2024@localhost:5988/ecommerce_orders/_design/audit/_view/conflicts_only_fast>

\# Vue optimisée sur EU1

curl -w \"\\nVue optimisée EU1: %{time_total}s\\n\"
<http://admin:ecommerce2024@localhost:5989/ecommerce_orders/_design/audit/_view/conflicts_only_fast>

\# Vue optimisée sur AP1

curl -w \"\\nVue optimisée AP1: %{time_total}s\\n\"
<http://admin:ecommerce2024@localhost:5990/ecommerce_orders/_design/audit/_view/conflicts_only_fast>

Équipe B :\
`curl`` ``-w`` ``"\``nTemps`` NA1: %{``time_``total``}s``\n"`` `<http://admin:ecommerce2024@localhost:5987/ecommerce_leads/_design/sales_audit/_view/synchronization_performance_optimized?reduce=true&group=true>

\# Vue optimisée sur NA2

`curl`` ``-w`` ``"\``nTemps`` NA2: %{``time_total``}s\n"`` ``"http://admin:ecommerce2024@localhost:5988/ecommerce_leads/_design/sales_audit/_view/synchronization_performance_optimized?reduce=true&group=true"`\
\# Vue optimisée sur EU1

`curl`` ``-w`` ``"\``nTemps`` EU1: %{``time_total``}s\n"`` ``"http://admin:ecommerce2024@localhost:5989/ecommerce_leads/_design/sales_audit/_view/synchronization_performance_optimized?reduce=true&group=true"`\
\# Vue optimisée sur AP1

`curl`` ``-w`` ``"\``nTemps`` AP1: %{``time_total``}s\n"`` ``"http://admin:ecommerce2024@localhost:5990/ecommerce_leads/_design/sales_audit/_view/synchronization_performance_optimized?reduce=true&group=true"`

### **Livrables Étape 3**

- **Analyse initiale** : Temps de réponse de la vue 5
- **Code d\'optimisation** : Index ou vue optimisée dans Fauxton
- **Comparaison** : Temps avant/après sur les 4 nœuds

  --------------------------------------------------------------------
  **Nœud**         **Vues           **Vues            **Gain (%)**
                   originales**     optimisées**      
  ---------------- ---------------- ----------------- ----------------
  **NA1**                                             

  **NA2**                                             

  **EU1**                                             

  **AP1**                                             

  **Moyenne**                                         
  --------------------------------------------------------------------

> **Formule gain** : ((Temps_original - Temps_optimisé) /
> Temps_original) × 100

- **Recommandations** : Améliorations pour registres distribués

## Livrables Finaux

### Étape 0 (25 points)

config.go complété (8 réplications)\
4 captures Fauxton avec bases créées\
Sortie de la commande verify\
Schéma d'architecture détaillé

### Étape 1 (30 points)

Tableaux statistiques de chargement\
4 exemples de documents JSON\
Tableau de mapping SQL → NoSQL\
Preuve de cohérence (curl ou captures)

### Étape 2 (35 points)

Logs d'exécution du scénario\
Code resolver.go complété\
5 captures Fauxton des vues\
Résultats des tests curl\
Rapport d'analyse des resultats

### Étape 3 (20 points)

Mesures de performance initiales\
Code/Index d'optimisation\
Tableau comparatif 4 nœuds\
Document de recommandations

**Durée estimée** : 2-3 semaines\
**Modalités** : Binômes avec démonstration incluant test de panne
