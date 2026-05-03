import os
import logging

logger = logging.getLogger(__name__)


class FraudGraphClient:
    """Neo4j-backed fraud graph queries.

    Gracefully degrades to zero-risk stubs when NEO4J_URL is not configured.
    """

    def __init__(
        self,
        url: str | None = None,
        user: str | None = None,
        password: str | None = None,
    ):
        self.url = url or os.getenv("NEO4J_URL", "")
        self.user = user or os.getenv("NEO4J_USER", "neo4j")
        self.password = password or os.getenv("NEO4J_PASSWORD", "")
        self._driver = None

        if self.url:
            try:
                from neo4j import GraphDatabase

                self._driver = GraphDatabase.driver(
                    self.url, auth=(self.user, self.password)
                )
                logger.info("Neo4j driver initialised: %s", self.url)
            except Exception as exc:
                logger.warning("Could not connect to Neo4j (%s) — using stubs: %s", self.url, exc)
                self._driver = None
        else:
            logger.warning("NEO4J_URL not set — using zero-risk stubs for fraud graph")

    def _session(self):
        if self._driver is None:
            raise RuntimeError("Neo4j driver not available")
        return self._driver.session()

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def get_vendor_risk(self, vendor_name: str) -> dict:
        """Return invoice count and risk score for a vendor."""
        if self._driver is None:
            return {"invoice_count": 0, "risk_score": 0.0}
        try:
            with self._session() as session:
                result = session.run(
                    """
                    MATCH (v:Vendor {name: $name})-[:ISSUED]->(i:Invoice)
                    RETURN count(i) AS invoice_count, coalesce(v.risk_score, 0.0) AS risk_score
                    """,
                    name=vendor_name,
                )
                record = result.single()
                if record:
                    return {
                        "invoice_count": record["invoice_count"],
                        "risk_score": float(record["risk_score"]),
                    }
            return {"invoice_count": 0, "risk_score": 0.0}
        except Exception as exc:
            logger.error("get_vendor_risk error: %s", exc)
            return {"invoice_count": 0, "risk_score": 0.0}

    def check_bank_account_sharing(self, vendor_name: str) -> bool:
        """Return True if this vendor shares a bank account with any other vendor."""
        if self._driver is None:
            return False
        try:
            with self._session() as session:
                result = session.run(
                    """
                    MATCH (v1:Vendor {name: $name})-[:PAID_TO]->(b:BankAccount)<-[:PAID_TO]-(v2:Vendor)
                    WHERE v1 <> v2
                    RETURN count(v2) > 0 AS shared
                    """,
                    name=vendor_name,
                )
                record = result.single()
                if record:
                    return bool(record["shared"])
            return False
        except Exception as exc:
            logger.error("check_bank_account_sharing error: %s", exc)
            return False

    def get_betweenness_centrality(self, vendor_name: str) -> float:
        """Return betweenness centrality for a vendor (uses risk_score as proxy)."""
        if self._driver is None:
            return 0.0
        try:
            with self._session() as session:
                # Try GDS first
                try:
                    result = session.run(
                        """
                        MATCH (v:Vendor {name: $name})
                        CALL gds.betweenness.stream('vendor-graph')
                        YIELD nodeId, score
                        WHERE gds.util.asNode(nodeId) = v
                        RETURN score
                        """,
                        name=vendor_name,
                    )
                    record = result.single()
                    if record:
                        return float(record["score"])
                except Exception:
                    pass  # GDS not available — fall through to proxy

                # Proxy: use risk_score
                result = session.run(
                    "MATCH (v:Vendor {name: $name}) RETURN coalesce(v.risk_score, 0.0) AS score",
                    name=vendor_name,
                )
                record = result.single()
                if record:
                    return float(record["score"])
            return 0.0
        except Exception as exc:
            logger.error("get_betweenness_centrality error: %s", exc)
            return 0.0

    def close(self):
        if self._driver:
            self._driver.close()
