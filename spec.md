======================================================================
Project Specification & Roadmap: Iara Personal AI Assistant
======================================================================
Version: 1.0
Date: August 31, 2025
Status: In Development (Phase 1)

----------------------------------------------------------------------
1. Project Description
----------------------------------------------------------------------

Iara is a hyper-personalized, self-hosted AI assistant designed to serve as an intelligent, context-aware agent. The primary goal is to create a secure, private, and powerful digital assistant that can be extended to manage personal information, automate tasks, and engage proactively with the user. The system is built on a modern, containerized, microservice-oriented architecture, with a core backend developed in Go.

----------------------------------------------------------------------
2. Technology Stack & Architectural Components
----------------------------------------------------------------------

The project utilizes a curated stack of modern technologies, each chosen for a specific role.

- Host Environment:
  - VPS: Debian Linux (TNAHosting 12GB RAM, 4vCPU, 500GB SSD)

- Core Stack:
  - Orchestration: Docker & Docker Compose for container management and deployment.
  - Core Backend API: A custom-built API in Go (Golang). This is the central logic hub of the assistant.
  - Event Gateway: n8n, running self-hosted. It serves as the bridge between external services (like Telegram) and the core Go API.
  - Vector Database (Memory): ChromaDB, for storing and retrieving textual information via semantic search (vector embeddings).
  - AI & Language Models: Google's Generative Language API, specifically:
    - `gemini-pro` for generative text and reasoning.
    - `embedding-001` for converting text into vector embeddings.
  - User Interface (Phase 1): Telegram Bot.

----------------------------------------------------------------------
3. Full Communication Flow
----------------------------------------------------------------------

The primary interaction pattern is a Retrieval-Augmented Generation (RAG) flow, orchestrated between the components.

1.  Input: The user sends a message to the Telegram bot.
2.  Gateway Trigger: The n8n `Telegram Trigger` node receives the message payload.
3.  API Call: n8n's `HTTP Request` node makes a POST request to a dedicated endpoint on the Go API (e.g., `http://go_api:8080/v1/message`), forwarding the user's text and metadata.
4.  Embedding Generation: The Go API receives the request. It makes an API call to the Google `embedding-001` model with the user's query text.
5.  Vector Retrieval: The Go API receives the resulting embedding vector from Google.
6.  Memory Query: The Go API sends this query vector to the ChromaDB API. ChromaDB performs a similarity search and returns the most relevant document(s) from its stored memory.
7.  Prompt Augmentation: The Go API constructs a new, detailed prompt for the generative model. This prompt contains the original user question augmented with the context retrieved from ChromaDB.
8.  Content Generation: The Go API sends this augmented prompt to the Google `gemini-pro` model.
9.  Response Reception: The Go API receives the final, context-aware text response from Gemini.
10. Gateway Response: The Go API returns this final text in its HTTP response to n8n.
11. Output: n8n's workflow completes, and its final `Telegram` node sends the text response back to the user in the original chat.

The "learn" flow is a subset of this, stopping at step 6 after storing the new fact's text and embedding in ChromaDB.

----------------------------------------------------------------------
4. Implementation Steps & Validation (Roadmap)
----------------------------------------------------------------------

The project is structured in distinct, deliverable phases.

- Phase 0: Security & Foundation (Completed)
  - Objective: Harden the VPS and establish a secure access method.
  - Deliverables: A fully updated Debian server, a personal `sudo` user, key-only SSH authentication, disabled root/password login, and an active firewall (UFW) with Fail2Ban protection.
  - Validation: Successful and secure login as a non-root user via SSH key.

- Phase 1: MVP Core (Current Phase)
  - Objective: Deploy the core infrastructure and implement the basic RAG loop.
  - Deliverables:
    1.  All services (n8n, ChromaDB, Go API) running and networked via Docker Compose.
    2.  A functional Go API skeleton with a health check endpoint.
    3.  Two core n8n workflows: "Learn Fact" and "Main Conversation".
    4.  Go API logic for the `/learn` command (embedding + storage).
    5.  Go API logic for the query command (embedding -> query -> generation -> response).
  - Validation: The user can successfully teach Iara a new fact via a Telegram command and then ask a question that gets answered using that stored fact.

- Phase 2: Function Calling & Tooling (Future)
  - Objective: Extend Iara's capabilities beyond Q&A to perform actions.
  - Deliverables: Integration with Google Calendar API. Iara will be able to create and manage calendar events based on natural language commands.
  - Validation: User can say "Schedule a meeting for tomorrow at 2 PM" and see the event appear in their Google Calendar.

- Phase 3: Proactive Engagement (Future)
  - Objective: Enable Iara to initiate conversations.
  - Deliverables: n8n workflows triggered by time (Cron jobs) that call the Go API to perform tasks like requesting an end-of-day summary from the user.
  - Validation: The user receives a daily message from Iara at a scheduled time.

----------------------------------------------------------------------
5. Core Features & Usability Expectations
----------------------------------------------------------------------

- Current Scope Features (Phase 1):
  - Fact Ingestion: The ability to add information to Iara's knowledge base using a specific command (e.g., `/learn`).
  - Fact Retrieval: The ability to ask natural language questions and receive answers based on the stored knowledge.

- Long-Term Features (Vision):
  - Task Automation: Seamlessly manage external services like calendars, to-do lists, etc.
  - Information Synthesis: Answer complex questions by combining information from its knowledge base and real-time data.
  - Proactive Assistance: Provide reminders, ask for daily updates, and offer suggestions based on user patterns.
  - Enhanced Modalities: Future integration of a voice interface (Speech-to-Text, Text-to-Speech).

- Example of Expected Usability (Car Maintenance Scenario):
  1.  User teaches Iara: `/learn Car model is Toyota Etios 2019. Last oil change was on April 12, 2025. Tire pressure should be 32 PSI.`
  2.  Later, user asks: `When was my last oil change?`
  3.  Iara responds: `Your last oil change was on April 12, 2025.`
  4.  User asks: `What's the recommended tire pressure for my car?`
  5.  Iara responds: `The recommended tire pressure for your Toyota Etios is 32 PSI.`

----------------------------------------------------------------------
6. Code Standards
----------------------------------------------------------------------

As the core backend is in Go, the project will adhere to professional development standards.

- Code Style: Idiomatic Go, enforced by `go fmt` and `go vet`.
- Architecture: Modular, with a clear separation of concerns (e.g., API handlers, business logic/services, external clients).
- Error Handling: Consistent and explicit error handling throughout the application.
- Configuration: No hardcoded secrets or values. Configuration will be managed via environment variables passed through `docker-compose.yml`.
- Testing: A commitment to writing unit tests for core logic and integration tests for API endpoints.
