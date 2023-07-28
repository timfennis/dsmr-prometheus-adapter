use std::sync::Arc;

use anyhow::Context;
use axum::{extract::State, routing::get, Router};
use axum_prometheus::{PrometheusMetricLayerBuilder, metrics_exporter_prometheus::PrometheusHandle};
use metrics::gauge;
use serde::Deserialize;
use serde_json::Value;
use tokio::signal::unix::{signal, SignalKind};
use url::Url;

#[derive(Deserialize, Debug)]
struct Config {
    dsmr_base_url: Url,
}

struct AppState {
    prometheus: PrometheusHandle,
    config: Config,
}

const METRICS_PREFIX: &str = "dsmr_logger";

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let config = envy::from_env::<Config>().context("cannot parse config from environment")?;

    let (metric_layer, metric_handle) = PrometheusMetricLayerBuilder::new()
        .with_ignore_patterns(&["/metrics", "/favicon.ico"])
        .with_prefix(METRICS_PREFIX)
        .with_default_metrics()
        .build_pair();

    let state = Arc::new(AppState {
        prometheus: metric_handle,
        config,
    });

    // build our application with a single route
    let app = Router::new()
        .route(
            "/metrics",
            get(metrics_handler),
        )
        .with_state(state)
        .layer(metric_layer);

    // run it with hyper on localhost:3000
    axum::Server::bind(&"0.0.0.0:8080".parse().unwrap())
        .serve(app.into_make_service())
        .with_graceful_shutdown(shutdown_signal())
        .await
        .context("failed to start webserver")
}

async fn metrics_handler(State(state): State<Arc<AppState>>) -> String {
    poll_logger(state.config.dsmr_base_url.clone()).await.unwrap();

    state.prometheus.render()
}

#[derive(Deserialize)]
struct Container {
    pub actual: Vec<Measurement>,
}

#[derive(Deserialize)]
struct Measurement {
    pub name: String,
    pub value: serde_json::Value,
    pub unit: Option<String>,
}

async fn poll_logger(base_url: Url) -> anyhow::Result<()> {
    let client = reqwest::Client::new();

    let mut url = base_url.clone();
    url.set_path("/api/v1/sm/actual");

    let data: Container = client
        .get(url)
        .send()
        .await
        .context("failed to send HTTP request")?
        .json()
        .await
        .context("failed to decode response into JSON")?;

    for m in data.actual {
        match m {
            Measurement {
                name,
                value: Value::Number(number),
                unit: Some(unit),
            } => {
                gauge!(
                    format!("{}_{}_{}", METRICS_PREFIX, name, unit.to_lowercase()),
                    number.as_f64().unwrap()
                );
            }

            Measurement {
                name,
                value: Value::Number(number),
                unit: None,
            } => {
                gauge!(
                    format!("{}_{}", METRICS_PREFIX, name),
                    number.as_f64().unwrap()
                );
            }
            _ => {
                // ignore
            }
        }
    }
    Ok(())
}

async fn shutdown_signal() {
    let mut sigterm = signal(SignalKind::terminate()).unwrap();

    tokio::select! {
        _ = sigterm.recv() => {
            println!("received sigterm, shutting down");
        }
        _ = tokio::signal::ctrl_c() => {
            println!("received ctrl-c, shutting down");
        }
    }
}
