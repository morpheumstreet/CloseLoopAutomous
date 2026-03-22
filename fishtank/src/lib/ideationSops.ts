/** Four distinct ideation SOPs — content mirrors the operator playbook (Mission/Vision first, alignment-weighted). */

export type SopAgendaRow = { time: string; phase: string; techniques: string; output: string };

export type SopEvalRow = { criterion: string; weight: string; description: string };

export type IdeationSopDefinition = {
  key: string;
  n: 1 | 2 | 3 | 4;
  shortTitle: string;
  fullTitle: string;
  covers: string;
  purpose: string;
  scope: string;
  participants: string;
  preparation: string;
  agenda: SopAgendaRow[];
  groundRules: string[];
  evaluation: SopEvalRow[];
  deliverables: string;
  cadenceTools: string;
};

export const IDEATION_SOPS: IdeationSopDefinition[] = [
  {
    key: 'creative',
    n: 1,
    shortTitle: 'Fast Creative & Content',
    fullTitle: 'SOP 1: Fast Creative & Content Workflow',
    covers:
      'Twitter/X lines, TikTok/Reels/Shorts, YouTube video, Short film, Documentation/interview video, Podcast, Newsletter series',
    purpose:
      'Generate 100–500+ high-engagement content ideas weekly that directly advance our Mission (core purpose today) and accelerate our Vision (future aspiration). Focus = virality, emotional resonance, and algorithm performance.',
    scope: 'All storytelling & social formats. Trigger: weekly content calendar or trend spike.',
    participants: 'Facilitator + content creator + brand voice guardian + data analyst + 1–2 outsiders (4–8 people).',
    preparation:
      'Display Mission & Vision. Write 3–5 HMW questions tied to them. Gather trends, comments, competitor hooks, audience data (1–2 hours before).',
    agenda: [
      {
        time: '5 min',
        phase: 'Kickoff',
        techniques: 'Read Mission/Vision + HMW',
        output: 'Alignment locked',
      },
      {
        time: '25–40 min',
        phase: 'Diverge',
        techniques: 'Crazy 8s, Hook Bank, 5×5×5 Pillars, Trend Jacking, AI variations',
        output: '100–300 raw ideas',
      },
      {
        time: '10 min',
        phase: 'Converge',
        techniques: 'Cluster + 3-dot silent vote',
        output: 'Top 15–25',
      },
      {
        time: '10 min',
        phase: 'Develop & Score',
        techniques: 'Flesh 1-sentence + thumbnail sketch',
        output: '5–10 ready concepts',
      },
    ],
    groundRules: ['Quantity > quality.', 'Defer judgment.', '“Yes, and…”.', 'Wild = encouraged.'],
    evaluation: [
      { criterion: 'Mission/Vision Alignment', weight: 'High', description: 'Directly advances Mission & Vision?' },
      { criterion: 'Predicted Engagement', weight: 'High', description: 'Hook strength, save/share potential' },
      { criterion: 'Novelty', weight: 'Med', description: 'Fresh angle or format?' },
      { criterion: 'Speed to Produce', weight: 'Med', description: 'Can we ship in <48h?' },
      { criterion: 'Brand Safety', weight: 'Med', description: 'Zero risk to reputation' },
    ],
    deliverables:
      'Top 5 concepts + calendar slots + owners. Export to content repo. A/B test winners live. Retro in 1 sentence (within 24h).',
    cadenceTools: 'Daily/weekly 45-min sprints. Tools: Miro/FigJam + CapCut + analytics.',
  },
  {
    key: 'analytical',
    n: 2,
    shortTitle: 'Analytical Insight & Edge',
    fullTitle: 'SOP 2: Analytical Insight & Edge Workflow',
    covers: 'Research discovery, Investment idea, Prediction/operation, Gamble/trading strategies, Macro/geopolitical thesis',
    purpose:
      'Uncover high-conviction insights, theses, and edges that move us toward our Vision while staying rooted in our Mission.',
    scope: 'All research, forecasting, and probabilistic strategy work.',
    participants:
      'Facilitator + domain expert + data/quant person + contrarian + Mission/Vision guardian (3–6 people).',
    preparation:
      'Display Mission & Vision. Define base-rate question + 3 HMW. Gather data sources, filings, literature gaps.',
    agenda: [
      {
        time: '10 min',
        phase: 'Frame',
        techniques: 'Mission/Vision + base rates',
        output: 'Clear question',
      },
      {
        time: '40 min',
        phase: 'Diverge',
        techniques: 'Gap analysis, analogies, scenario branching',
        output: '20–50 raw theses',
      },
      {
        time: '20 min',
        phase: 'Converge',
        techniques: 'Cluster + EV/premium scoring',
        output: 'Top 8–12',
      },
      {
        time: '20 min',
        phase: 'Deepen',
        techniques: 'Assumption mapping + probability adjust',
        output: '3–5 refined theses',
      },
    ],
    groundRules: ['Evidence first.', 'Contrarian views required.', 'No sacred cows.'],
    evaluation: [
      { criterion: 'Mission/Vision Alignment', weight: 'High', description: 'Advances Mission & Vision?' },
      { criterion: 'Edge Size / Mispricing', weight: 'High', description: 'Quantifiable advantage?' },
      { criterion: 'Evidence Strength', weight: 'High', description: 'Data + backtest support?' },
      { criterion: 'Risk/Reward Ratio', weight: 'Med', description: 'Downside protected?' },
      { criterion: 'Speed to Validate', weight: 'Med', description: 'Can we test in <30 days?' },
    ],
    deliverables:
      '3–5 written theses (1-pager each) + validation plan + owners. Upload to research repo.',
    cadenceTools: 'Weekly or bi-weekly. Tools: Notion + Excel/Google Sheets + simulation software.',
  },
  {
    key: 'digital_product',
    n: 3,
    shortTitle: 'Digital Product',
    fullTitle: 'SOP 3: Digital Product Workflow',
    covers: 'Engineered software product, No-code/low-code app',
    purpose:
      'Create user-loved digital products and features that fulfil our Mission today and scale our Vision tomorrow.',
    scope: 'All software, SaaS, apps, and no-code builds.',
    participants:
      'Facilitator + product owner + engineer + designer + user researcher + Mission/Vision champion (5–9 people).',
    preparation: 'Display Mission & Vision. Map 3–5 Jobs-to-be-Done + HMW tied to them.',
    agenda: [
      {
        time: '10 min',
        phase: 'Frame',
        techniques: 'Mission/Vision + JTBD',
        output: 'User outcome clear',
      },
      {
        time: '30 min',
        phase: 'Diverge',
        techniques: 'Crazy 8s, user-story mashups, forced combinations',
        output: '60–150 raw features',
      },
      {
        time: '15 min',
        phase: 'Converge',
        techniques: '2×2 matrix (Value vs Effort)',
        output: 'Top 12–20',
      },
      {
        time: '20 min',
        phase: 'Develop',
        techniques: '1-pager + painted-door test plan',
        output: '4–7 viable concepts',
      },
    ],
    groundRules: ['User-first.', 'Build to learn.', 'Fast > perfect.'],
    evaluation: [
      { criterion: 'Mission/Vision Alignment', weight: 'High', description: 'Directly supports both?' },
      { criterion: 'User Value / JTBD', weight: 'High', description: 'Solves real job?' },
      { criterion: 'Feasibility & Speed', weight: 'High', description: 'MVP in <2 weeks?' },
      { criterion: 'Scalability', weight: 'Med', description: 'Can grow with Vision?' },
      { criterion: 'Technical Risk', weight: 'Med', description: 'Low dependency risk?' },
    ],
    deliverables:
      'Top 4 concepts with user story, assumptions, and painted-door test owner. Log in Jira/Notion.',
    cadenceTools: 'Every sprint (2 weeks). Tools: FigJam + Figma + Bubble/Adalo (no-code).',
  },
  {
    key: 'hardware',
    n: 4,
    shortTitle: 'Hardware & Regulated Systems',
    fullTitle: 'SOP 4: Rigorous Hardware & Regulated Systems Workflow',
    covers:
      'IoT/connected device, Electronic hardware, Robot product, Medical product, Drugs/pharma, Biotech/gene, Aerospace',
    purpose:
      'Solve hard technical contradictions to deliver breakthrough physical/regulatory-compliant products that realise our Mission and Vision.',
    scope: 'All hardware, robotics, medical, pharma, biotech, and aerospace systems.',
    participants:
      'Facilitator + lead engineer + regulatory expert + clinician/scientist + Mission/Vision guardian (5–10 people).',
    preparation: 'Display Mission & Vision. List contradictions + 3 HMW framed around them.',
    agenda: [
      {
        time: '15 min',
        phase: 'Frame',
        techniques: 'Mission/Vision + contradictions',
        output: 'Problem locked',
      },
      {
        time: '45–60 min',
        phase: 'Diverge',
        techniques: 'TRIZ 40 Principles + Morphological Analysis',
        output: '30–80 raw solutions',
      },
      {
        time: '20 min',
        phase: 'Converge',
        techniques: 'Feasibility matrix + risk scoring',
        output: 'Top 8–12',
      },
      {
        time: '30–45 min',
        phase: 'Develop & Gate',
        techniques: 'Sketch + FMEA + regulatory checklist',
        output: '3–5 gated concepts',
      },
    ],
    groundRules: ['Physics & regulation are non-negotiable.', 'Safety first.', 'Document everything.'],
    evaluation: [
      { criterion: 'Mission/Vision Alignment', weight: 'High', description: 'Advances both without compromise?' },
      { criterion: 'Technical Feasibility', weight: 'High', description: 'Physics/regulatory viable?' },
      { criterion: 'Safety & Risk (FMEA)', weight: 'High', description: 'Acceptable risk level?' },
      { criterion: 'Innovation Level', weight: 'Med', description: 'Resolves contradiction uniquely?' },
      { criterion: 'Time-to-Prototype', weight: 'Med', description: 'First prototype in <3 months?' },
    ],
    deliverables:
      '3–5 concepts with TRIZ resolution table, risk register, and stage-gate plan. Upload to PLM/regulated repo.',
    cadenceTools:
      'Monthly deep sessions + stage-gate reviews. Tools: TRIZ software + CAD + simulation + Jira.',
  },
];

export const IDEATION_SOP_SYSTEM_STEPS = [
  'Pick the SOP that matches your category.',
  'Always start with Mission & Vision displayed.',
  'Run the session exactly as written.',
  'Only ideas that score high on Alignment move forward.',
] as const;
