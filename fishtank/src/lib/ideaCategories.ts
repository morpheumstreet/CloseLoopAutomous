/**
 * Ideation bucket — maps 1:1 to one of the four SOP standards (creative, analytical, digital product, hardware/regulated).
 * Slugs match arms domain.AllowedIdeaCategory (plus legacy MC keys still accepted server-side).
 */
export type IdeationSopIndex = 1 | 2 | 3 | 4;

export const IDEATION_BUCKETS = [
  { value: 'twitter_x_line', label: 'Twitter / X line (single post / thread ideas)', sop: 1 as const },
  { value: 'tiktok_reels_shorts', label: 'TikTok / Reels / Shorts ideation', sop: 1 as const },
  { value: 'youtube_video', label: 'YouTube video', sop: 1 as const },
  { value: 'short_film', label: 'Short film', sop: 1 as const },
  { value: 'documentation_video', label: 'Documentation video / Documentary-style', sop: 1 as const },
  { value: 'interview_video', label: 'Interview video', sop: 1 as const },
  { value: 'podcast_episode', label: 'Podcast episode', sop: 1 as const },
  { value: 'newsletter_series', label: 'Newsletter / long-form written series', sop: 1 as const },
  { value: 'research_discovery', label: 'Research discovery', sop: 2 as const },
  { value: 'investment_idea', label: 'Investment idea', sop: 2 as const },
  { value: 'prediction_forecasting', label: 'Prediction idea / operation / forecasting', sop: 2 as const },
  { value: 'gamble_betting_systems', label: 'Gamble strategies / betting systems', sop: 2 as const },
  { value: 'gamble', label: 'Gamble / gambling (general)', sop: 2 as const },
  { value: 'casino', label: 'Casino / operator / gaming floor ideas', sop: 2 as const },
  { value: 'engineered_software', label: 'Engineered software product', sop: 3 as const },
  { value: 'nocode_lowcode_product', label: 'No-code / low-code product', sop: 3 as const },
  { value: 'blockchain_protocol', label: 'Blockchain protocol', sop: 3 as const },
  { value: 'meme_coin', label: 'Meme coin / token concept', sop: 3 as const },
  { value: 'blockchain_smart_contract', label: 'Blockchain smart contract', sop: 3 as const },
  { value: 'crosschain', label: 'Cross-chain / interoperability', sop: 3 as const },
  { value: 'iot_device', label: 'IoT / connected device', sop: 4 as const },
  { value: 'electronic_hardware', label: 'Electronic hardware product', sop: 4 as const },
  { value: 'robot_product', label: 'Robot product', sop: 4 as const },
  { value: 'medical_device', label: 'Medical product / device', sop: 4 as const },
  { value: 'drugs_pharma', label: 'Drugs / pharma products', sop: 4 as const },
  {
    value: 'biotech_gene_aerospace_regulated',
    label: 'Biotech / gene therapy / aerospace-level regulated systems',
    sop: 4 as const,
  },
] as const;

export type IdeationBucketValue = (typeof IDEATION_BUCKETS)[number]['value'];

export const DEFAULT_IDEATION_BUCKET: IdeationBucketValue = IDEATION_BUCKETS[0].value;

export const IDEATION_SOP_NUMBERS: IdeationSopIndex[] = [1, 2, 3, 4];
