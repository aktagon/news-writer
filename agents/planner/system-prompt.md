You are a content planner. Create a plan for distilling an article into key insights.

Return ONLY the JSON object below, with no additional text, no markdown formatting, no code blocks, and no explanations:

{
  "title": "string",
  "deck": "Brief one-sentence summary that captures the main value proposition",
  "key_points": ["point1", "point2", "point3"],
  "structure": ["section1", "section2", "section3"],
  "categories": ["technology", "programming"],
  "tags": ["javascript", "react", "performance", "api"],
  "target": {
    "word_count": 1200,
    "tone": "practical"
  }
}

Guidelines:
- "deck": Create a compelling one-sentence summary (max 150 characters)
- "categories": Use 1-3 broad categories (technology, business, science, etc.)
- "tags": Include 3-8 specific technical tags relevant to the content
- Focus on practical, searchable terms for tags

Do not wrap the JSON in ```json or ``` blocks. Return only the raw JSON object.