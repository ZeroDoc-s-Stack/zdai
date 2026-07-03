---
name: worldview-scorer
description: 'A Bayesian-weighted adversarial scoring engine for evaluating and ranking worldviews, claims, philosophies, scriptures, ideologies, or any belief system. Extends the critical-analysis skill by resolving the tension between parsimony (fewer assumptions) and explanatory power through explicit mathematical weights and a forced mathematical verdict. Use this skill whenever the user wants to: score a worldview, compare competing worldviews, find which worldview wins against adversarial pressure, filter "weak" worldviews from "strong" ones, or understand which worldview is most epistemically defensible. Trigger on phrases like "score this worldview", "which worldview is stronger", "compare these beliefs", "is this a weak or strong worldview", "run the worldview filter", "bayesian worldview test", "which belief system holds up best", or any request to rank or evaluate belief systems against each other. Also trigger when the critical-analysis skill has already been run and the user wants a verdict with weights.'
---

# Worldview Scorer

## Purpose

This skill scores any worldview — a religion, philosophy, ideology, scientific paradigm, ethical system, or interpretive framework — against three classes of external force and produces a **mathematical verdict** resolving which worldview is stronger, weaker, or neutral.

It bridges the gap left by the 31 questions in the critical-analysis skill:

> **The unresolved tension:** Does a worldview that requires fewer assumptions beat one with greater explanatory power? The 31 questions surface both. This skill forces a weighted answer.

The answer is not purely logical and not purely Bayesian. It is both, applied in sequence. Logical priors are set first. Bayesian updating is applied against evidence with explicit caps and diminishing returns. Then a forced verdict is computed.

---

## Dependency

This skill extends the critical-analysis skill. Run Mode Detection from the critical-analysis skill first if you have not already done so. If you have already run the full critical-analysis rubric on the target text, use those outputs directly as inputs to Stage 0 of this skill.

If you are scoring a worldview without a specific text (a belief system named, not quoted), skip Mode Detection and proceed to Stage 0 directly.

---

## Architecture Overview

The scorer runs **six stages** and produces one composite score per worldview per dimension, then combines them into a final verdict.

```
Stage 0: Version Identification
  ↓
Stage 1: Dimension Extraction
  ↓
Stage 2: Prior Setting (Logical Weights)
  ↓
Stage 3: Evidence Intake (Adversarial / Anti-Adversarial / Neutral)
  ↓
Stage 4: Bayesian Updating
  ↓
Stage 5: Verdict Computation
```

When comparing two or more worldviews, run all six stages for each worldview, then run Stage 6: Comparative Verdict.

---

## Stage 0: Version Identification

_Which version of this worldview is being scored?_

**Goal:** Explicitly identify what form of the worldview is under evaluation before any scoring begins. This stage prevents the most common failure mode of the scorer: treating "Christianity" or "Buddhism" as monolithic when the version being scored determines the result more than any individual argument.

**Mandatory questions (answer all three):**

1. **Is this the canonical/dominant form, a specific historical articulation, or a text-based version?**
   - *Canonical*: The majority tradition's standard formulation (e.g., Nicene Christianity, Theravada Buddhism, Sunni Islam, Advaita Vedanta)
   - *Specific articulation*: A named school, thinker, or period within the tradition (e.g., Thomistic Christianity, Zen Buddhism, Mu'tazilite Islam)
   - *Text-based*: A specific document or argument being scored as a worldview (e.g., a blog post, a specific scripture, a philosophical treatise)

2. **What are the load-bearing claims of this specific version?** Name the 2–4 claims that, if falsified, would collapse this version of the worldview. These become the scoring targets — not the tradition's full doctrinal catalog.

3. **Does this version differ materially from the canonical form on any scored dimension?** If yes, flag each dimension affected and explain how. Results from non-canonical versions cannot be compared directly to canonical scores without this flag.

**Output of Stage 0:**

```
Version: [Canonical / Specific articulation / Text-based]
Load-bearing claims:
  1. [Claim]
  2. [Claim]
  [etc.]
Deviation from canonical form: [None / Yes — affected dimensions: ___]
Comparability note: [One sentence on whether this score can be compared
  to a canonical-form score of the same tradition]
```

---

## Stage 1: Dimension Extraction

_What are the load-bearing claims of this worldview on each scored dimension?_

**Goal:** Decompose the worldview into exactly **five scorable dimensions**. These are the axes on which evidence will be evaluated. Every worldview, no matter how complex, must compress into these five. Score the version identified in Stage 0 — not the canonical form unless Stage 0 determined this is the canonical form.

The five dimensions are fixed. Do not substitute them.

| Dimension | Definition | What "High" Looks Like |
|---|---|---|
| **P** — Parsimony | How many foundational assumptions does this worldview require before any claims can be made? Fewer assumptions = higher P. | One or two unfalsifiable priors. No additional hidden axioms. |
| **E** — Explanatory Power | How much of observable human experience, history, and natural phenomena does this worldview explain, account for, or make sense of? | Explains suffering, morality, consciousness, death, meaning, and social behavior without requiring ad hoc additions. |
| **C** — Coherence | Are the internal claims of the worldview mutually consistent? Does following its logic produce contradictions? | No internal contradictions. Claims do not undermine each other. |
| **D** — Durability | How has this worldview performed under sustained historical pressure — persecution, counter-evidence, rival traditions, scientific advancement? | Survives prolonged engagement. Adapts without collapsing. Its core claims are not refuted by subsequent findings. |
| **R** — Reproducibility | Can an independent agent — someone with no prior knowledge of this worldview — arrive at its core conclusions through reason, observation, or experience alone? | Core ethical and metaphysical conclusions are rediscovered independently across cultures. |

**Output of Stage 1:**

For each dimension:
- One sentence stating what this worldview specifically claims or does in that dimension
- A preliminary qualitative rating: High / Medium / Low / Unknown
- One sentence explaining the rating

---

## Stage 2: Prior Setting (Logical Weights)

_Before any evidence is applied, what do logic and structure alone suggest?_

**Goal:** Set **logical priors** for each dimension as a probability between 0.0 and 1.0. These are not Bayesian posteriors yet — they are what a careful reasoner would estimate before examining any external evidence, based solely on the internal structure of the worldview's claims.

**Prior-setting rules (apply in order):**

1. **Parsimony prior:** Count the number of foundational, unfalsifiable assumptions (axioms) the worldview requires. Apply:
   - 1 axiom: 0.85
   - 2 axioms: 0.70
   - 3 axioms: 0.55
   - 4 axioms: 0.40
   - 5+ axioms: 0.25
   - Note: An axiom is a claim that cannot be derived from simpler claims within the system and cannot be falsified. Distinguish axioms from claims that are derivable or testable.

2. **Explanatory Power prior:** Count the domains the worldview explicitly addresses (ethics, suffering, origins, death, meaning, social order, consciousness). Apply:
   - 6–7 domains: 0.80
   - 4–5 domains: 0.65
   - 2–3 domains: 0.50
   - 0–1 domain: 0.30
   - Penalize: if the worldview explains all domains only by adding new assumptions for each one, subtract 0.10 per ad hoc addition beyond the founding axioms.

3. **Coherence prior:** Run a quick internal consistency check. Apply:
   - No detected contradictions: 0.80
   - Minor tensions, resolvable within the tradition's own logic: 0.65
   - **Acknowledged paradoxes** — named tensions the tradition has formally recognized, argued about for generations, and built constructive theology/philosophy around: 0.55
   - **Active unresolved contradictions** — internal conflicts the tradition has not formally addressed or that produce logically incompatible claims: 0.35
   - Active contradictions the tradition denies exist: 0.20

   > **Critical distinction:** An acknowledged paradox (e.g., Trinity, no-self/karma, wave-particle duality) is epistemically different from an unresolved contradiction. A tradition that names its tension, produces centuries of engagement with it, and builds constructive thought around it is not in the same epistemic position as a tradition that simply contradicts itself unawares. Score accordingly.

4. **Durability prior:** Estimate based on age AND survival mechanism. Apply the base prior from age:
   - Worldview is < 100 years old with no significant counter-pressure: 0.40
   - Worldview is 100–500 years old, survived some pressure: 0.55
   - Worldview is 500–2000 years old, survived substantial pressure: 0.70
   - Worldview is 2000+ years old, survived sustained adversarial pressure: 0.80

   Then apply the **survival mechanism qualifier**:
   - Survived primarily through intellectual engagement, debate, and persuasion: no adjustment
   - Survived primarily through cultural inertia (default inheritance, no serious challenge): −0.10
   - Survived primarily through institutional/political power (state religion, enforced doctrine): −0.10
   - Survived primarily through military conquest or coercive enforcement: −0.15
   - Penalize: if the worldview's founding claims have been empirically falsified, subtract an additional 0.20.

   > These adjustments can stack. A worldview that is 2000 years old but survived primarily through state enforcement and cultural inertia may end up at 0.60 rather than 0.80.

5. **Reproducibility prior:** Estimate based on cross-cultural convergence. Apply:
   - Core conclusions rediscovered in 3+ independent traditions: 0.85
   - Core conclusions rediscovered in 1–2 independent traditions: 0.65
   - Core conclusions appear only in this tradition: 0.40
   - Core conclusions contradict what every independent tradition arrives at: 0.20

**Output of Stage 2:**

A prior probability table:

| Dimension | Prior (0.0–1.0) | Reasoning (one sentence) |
|---|---|---|
| P — Parsimony | ___ | |
| E — Explanatory Power | ___ | |
| C — Coherence | ___ | |
| D — Durability | ___ | |
| R — Reproducibility | ___ | |

**Pre-evidence composite:** Apply the same dimension weights used in Stage 5 (not a simple average). This makes the pre-evidence score directly comparable to the post-evidence score.

> Pre-evidence score = (P × 0.15) + (E × 0.25) + (C × 0.25) + (D × 0.20) + (R × 0.15)

---

## Stage 3: Evidence Intake

_What do adversarial, anti-adversarial, and neutral sources say?_

**Goal:** Collect evidence in three classes. Each class produces a **likelihood ratio** (LR) for each dimension — a multiplier expressing how much more likely the evidence is under this worldview than under the best available alternative.

Evidence classes:

| Class | Definition | Direction |
|---|---|---|
| **ADV** — Adversarial | Claims, traditions, findings, or worldviews that are explicitly opposed to this one, or that the worldview's claims directly contradict. | Pressure against. |
| **ANTI-ADV** — Anti-Adversarial | Claims, traditions, findings, or worldviews that independently arrive at the same or structurally equivalent conclusions. | Support for. |
| **NEU** — Neutral | Observations, data, or frameworks that have no stake in the worldview's truth but bear on its claims (empirical science, cross-cultural anthropology, historical record). | Neither for nor against — used to calibrate. |

**For each dimension, collect at minimum:**
- 1 ADV source or claim
- 1 ANTI-ADV source or claim
- 1 NEU source or datum

**Likelihood ratio assignment rules:**

For each piece of evidence, assign a raw likelihood ratio (LR):

| Evidence strength | LR range | Description |
|---|---|---|
| Decisive | 10–15 | The evidence is extremely difficult to explain without this dimension being true/false |
| Strong | 3–9 | The evidence significantly favors or disfavors this dimension |
| Moderate | 1.5–2.9 | The evidence leans one way but is consistent with alternatives |
| Weak | 1.0–1.4 | The evidence barely moves the needle |

For ADV evidence: the LR is applied as a **divisor** (it reduces the posterior).
For ANTI-ADV evidence: the LR is applied as a **multiplier** (it increases the posterior).
For NEU evidence: see NEU rules below.

**Diminishing returns rule:** Each additional ADV or ANTI-ADV argument targeting the same dimension gets a **0.8× discount** applied to its raw LR before multiplication. This prevents argument quantity from substituting for argument quality.

> Example: Three ANTI-ADV arguments for dimension E with raw LRs of 3.0, 2.5, and 2.0.
> Adjusted LRs: 3.0 (first, no discount), 2.0 (2.5 × 0.8), 1.28 (2.0 × 0.64).
> Net ANTI-ADV LR = 3.0 × 2.0 × 1.28 = 7.68 — not 15.0 as raw multiplication would give.

**NEU evidence rules:**
- If NEU evidence bears directly on **Explanatory Power (E) or Reproducibility (R)**, it may update the posterior by up to ±0.20.
- If NEU evidence bears on **Parsimony (P), Coherence (C), or Durability (D)**, it may update the posterior by up to ±0.10.
- If truly neutral (LR = 1.0), no update.
- NEU adjustments are applied after the Bayesian update, as flat additions or subtractions to the posterior, then re-clamped.

**Output of Stage 3:**

For each dimension, a table:

| Source | Class | Raw LR | Discount | Adj. LR | Dimension(s) affected | One-sentence rationale |
|---|---|---|---|---|---|---|
| [Source name or tradition] | ADV/ANTI-ADV/NEU | [value] | [×1.0 / ×0.8 / ×0.64...] | [value] | [P/E/C/D/R] | |

Minimum: 3 sources × 5 dimensions = 15 evidence rows. Quality over quantity — a decisive source counts more than five weak ones.

---

## Stage 4: Bayesian Updating

_Combine priors with evidence to produce posterior probabilities._

**Goal:** For each dimension, update the logical prior using the evidence collected in Stage 3.

**Step 1 — Apply the LR cap:** Before running the Bayesian formula, cap the net ADV and net ANTI-ADV LR products per dimension at **15.0**. If the product of adjusted LRs exceeds 15.0, use 15.0.

> This prevents three moderate arguments from producing the same pressure as one decisive argument, which would be epistemically unjustified.

**Step 2 — Bayesian update:**

```
Posterior = (Prior × Net_ANTI-ADV_LR) /
            (Prior × Net_ANTI-ADV_LR + (1 - Prior) × Net_ADV_LR)
```

Where Net_ANTI-ADV_LR and Net_ADV_LR are the diminishing-returns-adjusted, capped products from Stage 3. If no ADV evidence was collected for a dimension, Net_ADV_LR = 1.0. If no ANTI-ADV evidence, Net_ANTI-ADV_LR = 1.0.

**Step 3 — Apply NEU adjustment:** Apply the NEU flat adjustment (±0.20 max for E/R, ±0.10 max for P/C/D) to the posterior from Step 2.

**Step 4 — Clamp:** All posteriors must remain in [0.05, 0.95]. A worldview cannot be proven or disproven to certainty by this method.

**Output of Stage 4:**

| Dimension | Prior | Net ANTI-ADV LR (capped) | Net ADV LR (capped) | Raw Posterior | NEU adj. | Final Posterior |
|---|---|---|---|---|---|---|
| P — Parsimony | | | | | | |
| E — Explanatory Power | | | | | | |
| C — Coherence | | | | | | |
| D — Durability | | | | | | |
| R — Reproducibility | | | | | | |

---

## Stage 5: Verdict Computation

_Force a mathematical winner._

**Goal:** Compute the final weighted composite score. This score determines whether the worldview is **Strong**, **Moderate**, **Weak**, or **Neutral** on the scoring scale.

### Dimension Weights

Not all dimensions carry equal weight. The weights below resolve the parsimony vs. explanatory power tension directly by encoding their relative epistemic importance:

| Dimension | Weight | Rationale |
|---|---|---|
| P — Parsimony | **0.15** | Necessary but insufficient alone. A worldview with fewer assumptions is not automatically more true — it may simply explain less. |
| E — Explanatory Power | **0.25** | The primary criterion of a worldview's utility. A system that cannot account for the full range of human experience is deficient regardless of its elegance. |
| C — Coherence | **0.25** | Internal contradiction is fatal. A worldview that refutes itself is not a worldview. Tied with E because coherence gates the entire system. |
| D — Durability | **0.20** | Historical stress-testing is evidence of fitness. Survival under sustained adversarial pressure is the closest thing to empirical testing available for non-empirical claims. |
| R — Reproducibility | **0.15** | Independent convergence is the gold standard of non-empirical truth-tracking. But absence of independent convergence does not falsify — it just lowers confidence. |

**Weights sum to 1.00.**

### Composite Score Formula

```
Score = (P_post × 0.15) + (E_post × 0.25) + (C_post × 0.25) + (D_post × 0.20) + (R_post × 0.15)
```

### Verdict Scale

| Score | Verdict | Meaning |
|---|---|---|
| 0.80 – 0.95 | **STRONG** | This worldview holds up well against sustained pressure on all dimensions. Its weaknesses are minor or bridgeable. It deserves serious epistemic consideration. |
| 0.65 – 0.79 | **MODERATE-STRONG** | This worldview holds in most dimensions. It has one or two significant vulnerabilities that are not fatal. It should not be dismissed without engaging those vulnerabilities. |
| 0.50 – 0.64 | **MODERATE** | This worldview holds partially. It has real assets and real liabilities in roughly equal measure. Its strength is dimension-dependent. |
| 0.35 – 0.49 | **MODERATE-WEAK** | This worldview fails more than it holds. It may have one strong dimension that makes it compelling, but that strength is insufficient to support the overall structure. |
| 0.05 – 0.34 | **WEAK** | This worldview fails on most dimensions under sustained pressure. It should not be accepted without radical reconstruction of its core claims. |

**Output of Stage 5:**

```
Composite Score: [value]
Verdict: [STRONG / MODERATE-STRONG / MODERATE / MODERATE-WEAK / WEAK]

Dimension breakdown:
  P — Parsimony:        [posterior] × 0.15 = [weighted contribution]
  E — Explanatory Power:[posterior] × 0.25 = [weighted contribution]
  C — Coherence:        [posterior] × 0.25 = [weighted contribution]
  D — Durability:       [posterior] × 0.20 = [weighted contribution]
  R — Reproducibility:  [posterior] × 0.15 = [weighted contribution]
                                              ─────────────────────
  Composite:                                  [sum]

Strongest dimension: [name] at [posterior]
Weakest dimension:   [name] at [posterior]

The worldview's verdict is [VERDICT] because [one sentence naming the decisive factor].
```

---

## Stage 6: Comparative Verdict (Multi-Worldview Only)

_When two or more worldviews have been scored, force a ranking._

**Goal:** Run Stages 0–5 for each worldview independently (using the same evidence pool where the same evidence applies). Then rank them.

**Rules for comparison:**

1. **No tie-breaking by personal preference.** If two worldviews are within **0.05** of each other, declare a **statistical tie** and name the single dimension that would resolve it if more evidence were available.

2. **Version comparability check.** Before ranking, confirm that all worldviews being compared were scored at the same version type (canonical vs. specific articulation vs. text-based). If versions differ, note this explicitly — cross-version comparisons are valid but must be labeled.

3. **Identify the axis of divergence.** Name the one dimension where the worldviews differ most. This is usually where the real debate lives.

4. **State the decision boundary.** Identify: what piece of evidence, if established, would flip the ranking? This is the most useful output for a genuine truth-seeker.

**Output of Stage 6:**

```
Version comparability: [All same type / Mixed — see notes]

Ranked worldviews:
  1. [Worldview A] — Score: [X], Verdict: [VERDICT]
  2. [Worldview B] — Score: [X], Verdict: [VERDICT]
  [etc.]

Primary axis of divergence: [Dimension name]
  [Worldview A] scores [X] on [Dimension]; [Worldview B] scores [X]
  The gap is [large/moderate/small]: [one sentence on what drives it]

Decision boundary:
  If [specific claim or evidence] were established, [Worldview B] would
  overtake [Worldview A] on [Dimension], shifting the composite by approximately [delta].

Statistical tie (gap < 0.05): [Yes / No]
  [If yes: name the dimension and evidence type that would resolve it]
```

---

## Parsimony vs. Explanatory Power: The Resolved Tension

The critical-analysis rubric surfaces both without resolving the tradeoff. This skill resolves it as follows:

**The tradeoff is real but asymmetric.**

- Parsimony (P, weight 0.15) is a *tiebreaker*, not a primary criterion. Two worldviews equal on E, C, D, R should prefer the more parsimonious one. But parsimony alone cannot compensate for explanatory failure.
- Explanatory Power (E, weight 0.25) is the primary criterion because a worldview that cannot account for the full range of experience has failed its core function, regardless of how elegantly it is structured.
- The weight ratio 0.25:0.15 (E:P) encodes the answer: explanatory power outweighs parsimony by a factor of 5:3. This is a deliberate design choice grounded in the observation that every historically durable worldview explains more than its rivals, while no worldview survives on parsimony alone.

**The coherence weight (C, 0.25) acts as a gate.** A worldview with low coherence cannot score well even with high E and D, because incoherence undermines the reliability of every other dimension. However, the acknowledged paradox distinction in Stage 2 ensures that traditions with formally recognized, constructively engaged tensions are not penalized as harshly as traditions with ignored contradictions.

---

## Output Summary Template

After running all stages, produce this final summary:

```
═══════════════════════════════════════════════════════════
WORLDVIEW SCORE REPORT
Subject: [Worldview name or text title]
Version: [Canonical / Specific articulation / Text-based]
═══════════════════════════════════════════════════════════

PRE-EVIDENCE COMPOSITE:  [weighted, comparable to post-evidence]
POST-EVIDENCE COMPOSITE: [value]
FINAL VERDICT: [STRONG / MODERATE-STRONG / MODERATE / MODERATE-WEAK / WEAK]

DIMENSION SCORES (Posterior → Weighted)
  P — Parsimony:         [post] → [weighted]   [STRONG/MODERATE/WEAK]
  E — Explanatory Power: [post] → [weighted]   [STRONG/MODERATE/WEAK]
  C — Coherence:         [post] → [weighted]   [STRONG/MODERATE/WEAK]
  D — Durability:        [post] → [weighted]   [STRONG/MODERATE/WEAK]
  R — Reproducibility:   [post] → [weighted]   [STRONG/MODERATE/WEAK]

KEY ADVERSARIAL PRESSURE: [One sentence — what is the most damaging challenge?]
KEY ANTI-ADVERSARIAL SUPPORT: [One sentence — what is the most compelling corroboration?]
NEUTRAL VERDICT: [One sentence from a position with no stake in the outcome]

WHAT WOULD CHANGE THIS SCORE:
  To improve: [One specific claim or evidence that would raise the score]
  To worsen:  [One specific finding that would reduce the score]

FILTER RESULT:
  [PASS — worldview is epistemically defensible and worth sustained engagement]
  [CONDITIONAL PASS — passes on most dimensions; to move to PASS, must resolve: ___]
  [FAIL — worldview fails on dominant dimensions; not epistemically defensible without reconstruction]
═══════════════════════════════════════════════════════════
```

---

## Changelog from v1

These changes were made after observing the scorer run on five worldviews across two full passes with an expanded argument pool:

| Fix | What it addresses |
|---|---|
| Stage 0 added | Version identification prevents the largest single source of score variance — scoring "Christianity" without specifying which version |
| Acknowledged paradox tier in Coherence prior | Prevents traditions that formally engage their tensions from being penalized identically to traditions that ignore theirs |
| Survival mechanism qualifier in Durability prior | Removes bias toward old worldviews that survived primarily through political or military enforcement |
| Diminishing returns rule (0.8× discount per additional argument per dimension) | Prevents argument quantity from substituting for argument quality |
| LR product cap at 15.0 per dimension | Prevents cascading LR multiplication from collapsing a dimension to floor |
| NEU evidence expanded to ±0.20 for E/R | Allows genuinely important neutral evidence (fine-tuning, consciousness, cross-cultural ethics) to do meaningful work |
| Pre-evidence composite now uses dimension weights | Makes pre/post comparison meaningful — both scores are now on the same scale |
| Statistical tie threshold widened to 0.05 | Reflects actual scorer precision given LR judgment variance |
| CONDITIONAL PASS now names the specific resolving condition | Makes the filter output actionable rather than decorative |

---

## Notes on Non-Bias

This scoring system does not favor religious worldviews over secular ones, or empirical worldviews over metaphysical ones. The dimensions are structurally neutral:

- A scientific paradigm can score low on Parsimony (many auxiliary hypotheses) and high on Explanatory Power.
- A religious worldview can score high on Durability and Reproducibility and low on Coherence.
- A philosophical system can score high on Coherence and low on Durability.

The weights encode epistemic values (explanatory power and coherence matter more than elegance), not cultural or ideological preferences.

The evidence classes (ADV, ANTI-ADV, NEU) require the scorer to engage the strongest possible opposition and the strongest possible support. Weak adversarial evidence produces a falsely high score. The scorer must find the most damaging possible critique, not merely a convenient one.

For additional reference on the 31 questions used in the underlying critical analysis framework, see the critical-analysis skill.
