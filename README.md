# Sistema de Temperatura por CEP com OpenTelemetry e Zipkin

Este projeto implementa um sistema distribuído em Go que consulta o CEP e retorna informações de temperatura, instrumentado com **OpenTelemetry (OTEL)** e **Zipkin** para rastreamento distribuído.

## 📋 Visão Geral

O sistema é composto por:

- **Serviço A**: Recebe requisições CEP e valida o formato
- **Serviço B**: Processa CEP, busca localização e temperatura
- **OpenTelemetry Collector**: Coleta traces dos serviços
- **Zipkin**: Visualiza traces distribuídos
- **Prometheus**: Coleta métricas

## 🏗️ Arquitetura

```
Cliente → Serviço A → Serviço B → APIs Externas
                ↓         ↓
            OTEL Collector
                ↓
              Zipkin
```

### Fluxo de Dados:
1. **Serviço A** recebe CEP via POST
2. Valida formato (8 dígitos)
3. Encaminha para **Serviço B**
4. **Serviço B** consulta:
   - ViaCEP API (dados da localização)
   - OpenWeatherMap API (coordenadas + clima)
5. Retorna temperaturas em Celsius, Fahrenheit e Kelvin

## 🚀 Como Executar

### Pré-requisitos
- Docker & Docker Compose
- API Key do OpenWeatherMap

### 1. Clone e Configure

```bash
git clone <repo-url>
cd go-telemetry
```

### 2. Configure Variáveis de Ambiente

Crie um arquivo `.env`:

```bash
OPENWEATHERMAP_API_KEY=sua_api_key_aqui
```

> 📝 **Como obter API Key**: Acesse [WeatherAPI](https://openweathermap.org/api) e crie uma conta gratuita.

### 3. Execute com Docker Compose

```bash
docker-compose up --build
```

Isso iniciará:
- **Serviço A** em `http://localhost:8080`
- **Serviço B** em `http://localhost:8081`
- **Zipkin** em `http://localhost:9411`
- **Prometheus** em `http://localhost:9090`
- **OTEL Collector** em `localhost:4317` (gRPC)

## 📊 Testando o Sistema

### Requisição de Exemplo

```bash
curl -X POST http://localhost:8080 \
  -H "Content-Type: application/json" \
  -d '{"cep": "01310100"}'
```

### Resposta Esperada

```json
{
  "city": "São Paulo",
  "temp_C": 22.5,
  "temp_F": 72.5,
  "temp_K": 295.65
}
```

### Cenários de Teste

```bash
# ✅ CEP válido
curl -X POST http://localhost:8080 -H "Content-Type: application/json" -d '{"cep": "01310100"}'

# ❌ CEP inválido (422)
curl -X POST http://localhost:8080 -H "Content-Type: application/json" -d '{"cep": "123"}'

# ❌ CEP não encontrado (404)
curl -X POST http://localhost:8080 -H "Content-Type: application/json" -d '{"cep": "00000000"}'
```

## 🔍 Observabilidade com OTEL + Zipkin

### O que é OpenTelemetry?
OpenTelemetry é um framework de observabilidade que coleta traces, métricas e logs de aplicações distribuídas.

### O que é Zipkin?
Zipkin é uma ferramenta de rastreamento distribuído que visualiza como as requisições fluem entre serviços.

### Visualizando Traces

1. Acesse **Zipkin**: http://localhost:9411
2. Clique em "Run Query" para ver traces
3. Clique em um trace para ver detalhes

### Spans Implementados

#### Serviço A:
- `handle_cep_request`: Span principal da requisição
- `validate_cep`: Validação do formato CEP
- `forward_to_service_b`: Comunicação com Serviço B

#### Serviço B:
- `handle_temperature_request`: Span principal
- `validate_cep`: Validação CEP
- `fetch_viacep_data`: Busca dados ViaCEP
- `fetch_coordinates`: Busca coordenadas
- `fetch_weather_data`: Busca dados meteorológicos

### Atributos dos Spans

Cada span contém informações detalhadas:

```
- cep: CEP consultado
- city: Cidade encontrada
- state: Estado
- geo.latitude/longitude: Coordenadas
- temperature.*: Temperaturas
- http.status_code: Status HTTP
- api.name: Nome da API externa
```

## 📈 Métricas com Prometheus

Acesse: http://localhost:9090

Métricas disponíveis:
- Duração das requisições HTTP
- Contadores de spans
- Status codes
- Latência das APIs externas

## 🛠️ Desenvolvimento

### Estrutura do Projeto

```
├── cmd/
│   ├── service-a/         # Serviço A
│   └── service-b/         # Serviço B
├── internal/
│   ├── service-a/         # Lógica Serviço A
│   └── service-b/         # Lógica Serviço B
├── shared/
│   ├── telemetry/         # Configuração OTEL
│   ├── models/           # Modelos compartilhados
│   └── http-utils/       # Utilitários HTTP
├── configs/              # Configurações
├── docker-compose.yml    # Orquestração
└── otel-collector-config.yaml
```

### Executando Localmente (Desenvolvimento)

```bash
# Terminal 1 - OTEL Collector + Zipkin
docker-compose up zipkin otel-collector prometheus

# Terminal 2 - Serviço B
export OPENWEATHERMAP_API_KEY=sua_key
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
go run cmd/service-b/main.go

# Terminal 3 - Serviço A
export SERVICE_B_URL=http://localhost:8081
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
go run cmd/service-a/main.go
```

## 🔧 Configurações

### Variáveis de Ambiente

| Variável | Descrição | Padrão |
|----------|-----------|---------|
| `OPENWEATHERMAP_API_KEY` | API Key WeatherMap | - |
| `SERVICE_B_URL` | URL do Serviço B | `http://localhost:8081` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Endpoint OTEL | `otel-collector:4317` |

### Portas Utilizadas

| Serviço | Porta | Descrição |
|---------|-------|-----------|
| Serviço A | 8080 | API Principal |
| Serviço B | 8081 | API Interna |
| Zipkin | 9411 | Interface Web |
| Prometheus | 9090 | Métricas |
| OTEL Collector | 4317 | gRPC Receiver |
| OTEL Collector | 4318 | HTTP Receiver |

## 🚨 Troubleshooting

### Problemas Comuns

1. **Erro: "OPENWEATHERMAP_API_KEY environment variable is required"**
   - Solução: Configure a API key no `.env`

2. **Traces não aparecem no Zipkin**
   - Verifique se todos os containers estão rodando
   - Confirme se OTEL Collector está acessível

3. **Timeout nas APIs externas**
   - Verifique conectividade com internet
   - Confirme API key válida

### Logs Úteis

```bash
# Ver logs do OTEL Collector
docker-compose logs otel-collector

# Ver logs dos serviços
docker-compose logs service-a service-b
```

## 📚 APIs Utilizadas

- **ViaCEP**: https://viacep.com.br/ (consulta CEP)
- **OpenWeatherMap**: https://openweathermap.org/ (coordenadas + clima)

## 🎯 Funcionalidades de Observabilidade

### ✅ Implementado

- [x] Traces distribuídos entre serviços
- [x] Spans detalhados para cada operação
- [x] Propagação de contexto entre serviços
- [x] Instrumentação HTTP automática
- [x] Métricas customizadas
- [x] Atributos detalhados nos spans
- [x] Tratamento de erros com spans
- [x] Visualização no Zipkin

### 🔄 Possíveis Melhorias

- [ ] Logs estruturados correlacionados
- [ ] Métricas customizadas de negócio
- [ ] Alertas baseados em SLI/SLO
- [ ] Sampling inteligente
- [ ] Dashboard Grafana