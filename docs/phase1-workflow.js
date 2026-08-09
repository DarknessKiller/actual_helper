export const meta = {
  name: 'privacy_first_phase1',
  description: 'Phase 1 Foundation: Create TypeScript utilities, workers, provider parsers, and frontend integration for client-side privacy-first conversion.'
}

const ROOT = args.projectRoot
const FE = ROOT + '/frontend/src/lib'

// Phase 1A: Utility layer
phase('Utilities')

const [rulesResult, csvResult, dateutilResult, cardutilResult] = await parallel([
  () => agent(
    'Create the rule engine at ' + FE + '/rules.ts.\n\n' +
    'Port from Go internal/rule/engine.go. The rule engine has:\n' +
    '- shouldSkip(description): if includeKeywords non-empty, description must match one to pass; otherwise skip if matches excludeKeywords\n' +
    '- matchCategory(description): first-match wins, case-insensitive contains check against categories[].keyword\n\n' +
    'Input types already exist in types.ts: CategoryRule { keyword, group, category } and ProviderConfig { exclude_keywords, include_keywords, categories, account_mappings }.\n\n' +
    'Export createEngine(config: ProviderConfig) returning { shouldSkip, matchCategory }.\n' +
    'Export mergeConfigs(global: ProviderConfig, provider: ProviderConfig) => ProviderConfig concatenating arrays, provider mappings override global.',
    { description: 'Create rule engine', subagent_type: 'general-purpose', model: 'haiku' }
  ),

  () => agent(
    'Create the CSV serializer at ' + FE + '/csv.ts.\n\n' +
    'Export toActualCSV(reports: ActualBudgetReport[]): string and toActualCSVBlob(reports: ActualBudgetReport[]): Blob.\n' +
    'Header order: Account, Date, Payee, Notes, Category_Group, Category, Amount, Split_Amount, Cleared.\n' +
    'Use proper CSV escaping. Import ActualBudgetReport from ./types.',
    { description: 'Create CSV serializer', subagent_type: 'general-purpose', model: 'haiku' }
  ),

  () => agent(
    'Create the date utility at ' + FE + '/dateutil.ts.\n\n' +
    'Export: formatDate(ddmmm, stmtDate) for "DD MMM" to "YYYY-MM-DD" with year inference.\n' +
    'parseTNGDate(raw) for "M/D/YYYY". parseDayMonthYear(raw) for "D January 2006" or "D Jan 2006".\n' +
    'parseStatementDate(raw) for "02 Jan 2006" or "02 Jan 06".\n' +
    'truncate(s, n) for string truncation with "...".',
    { description: 'Create date utility', subagent_type: 'general-purpose', model: 'haiku' }
  ),

  () => agent(
    'Create the card utility at ' + FE + '/cardutil.ts.\n\n' +
    'Export normalizeWhitespace(s), extractAfterMarker(text, marker, fallback),\n' +
    'extractNearCardType(text, cardTypes, fallback), applyMapping(mapping, name).\n' +
    'Card number regex: /(\\d{4}[\\s-]*\\d{4}[\\s-]*\\d{4}[\\s-]*\\d{4})/.',
    { description: 'Create card utility', subagent_type: 'general-purpose', model: 'haiku' }
  ),
])

// Phase 1B: Worker wrappers
phase('Workers')

const [pdfResult, ocrResult] = await parallel([
  () => agent(
    'Create the PDF.js wrapper at ' + FE + '/pdf-worker.ts.\n\n' +
    'Import pdfjs-dist. Set workerSrc to pdfjs-dist/build/pdf.worker.min.mjs.\n' +
    'Export extractPDFText(file, password?, onProgress?) returning concatenated text from all pages.\n' +
    'Export extractPDFPageCount(file, password?) returning page count.',
    { description: 'Create PDF.js wrapper', subagent_type: 'general-purpose', model: 'haiku' }
  ),

  () => agent(
    'Create the Tesseract.js wrapper at ' + FE + '/ocr-worker.ts.\n\n' +
    'Export initOCR(langs?, onProgress?), ocrImage(imageData, onProgress?),\n' +
    'ocrPDFPages(file, password?, onProgress?), terminateOCR().\n' +
    'Use Tesseract.createWorker with cached singleton. For PDF pages use pdfjs-dist to render to canvas.',
    { description: 'Create Tesseract.js wrapper', subagent_type: 'general-purpose', model: 'haiku' }
  ),
])

// Phase 1C: Provider parsers
phase('Provider Parsers')

const [tngResult, rytResult, hlbResult, hsbcResult, uobResult, gxResult] = await parallel([
  () => agent(
    'Create TNG provider at ' + FE + '/providers/tng.ts.\n\n' +
    'Export parseTNG(text, config) returning ActualBudgetReport[].\n' +
    'Find "TNG WALLET TRANSACTION" with lastIndexOf. Split blocks by date regex.\n' +
    'Block lines: date, status, transactionType, reference, description, amount.\n' +
    'isCredit for Reload, Receive from Wallet, DUITNOW_RECEIVEFROM, Refund, GO+ Daily Earnings, GO+ Cash In.\n' +
    'trimAtReference stops at reference tokens (10+ digits, letter+digit patterns, TNGD/TNGQR/TNGOW prefixes).',
    { description: 'Create TNG parser', subagent_type: 'general-purpose', model: 'haiku' }
  ),

  () => agent(
    'Create RYT provider at ' + FE + '/providers/ryt.ts.\n\n' +
    'Export parseRYT(text, config) returning ActualBudgetReport[].\n' +
    'Find "Account Transactions" then "Baki" header. Extract account name.\n' +
    'Split blocks by date regex. Each block: first line date+desc, last line signed amount.\n' +
    'Skip "opening balance".',
    { description: 'Create RYT parser', subagent_type: 'general-purpose', model: 'haiku' }
  ),

  () => agent(
    'Create HLB provider at ' + FE + '/providers/hlb.ts.\n\n' +
    'Export parseHLB(text, config) returning ActualBudgetReport[].\n' +
    'DetectFormat: credit if Credit Card Number/HLB Credit Card/Tarikh Penyata. debit if A/C No/No Akaun/Deposit+Withdrawal.\n' +
    'Credit: statement date regex, transaction line regex, CR suffix = credit.\n' +
    'Debit layout: column-based with Deposit/Withdrawal headers. Debit columnar fallback.\n' +
    'Date format DD-MM-YYYY for debit.',
    { description: 'Create HLB parser', subagent_type: 'general-purpose', model: 'haiku' }
  ),

  () => agent(
    'Create HSBC provider at ' + FE + '/providers/hsbccredit.ts.\n\n' +
    'Export parseHSBCCredit(text, config) returning ActualBudgetReport[].\n' +
    'Account from Card Number marker. Statement date regex.\n' +
    'Transaction line: PostDate TransDate Description Amount CR.\n' +
    'Strip pipes/brackets. Skip summary prefixes.',
    { description: 'Create HSBC parser', subagent_type: 'general-purpose', model: 'haiku' }
  ),

  () => agent(
    'Create UOB provider at ' + FE + '/providers/uobcredit.ts.\n\n' +
    'Export parseUOBCredit(text, config) returning ActualBudgetReport[].\n' +
    'Statement date with 2 or 4 digit year. Transaction line regex with CR suffix.\n' +
    'Account from card type indicators (MASTERCARD, VISA). Skip summary patterns.',
    { description: 'Create UOB parser', subagent_type: 'general-purpose', model: 'haiku' }
  ),

  () => agent(
    'Create GXBank provider at ' + FE + '/providers/gxbank.ts.\n\n' +
    'Export parseGXBank(text, config) returning ActualBudgetReport[].\n' +
    'Account from "Statements of Accounts" header. Statement year from month+year line.\n' +
    'State machine after "Closing balance (RM)": date, time, desc(s), amount, balance.\n' +
    'Amount with +/- prefix. IsCredit = starts with "+".',
    { description: 'Create GXBank parser', subagent_type: 'general-purpose', model: 'haiku' }
  ),
])

// Phase 1D: Frontend integration
phase('Frontend Integration')

const integrationResult = await agent(
  'Update the frontend to use local conversion.\n\n' +
  '1. Replace ' + FE + '/api.ts entirely. New api.ts imports all provider parsers, pdf-worker, ocr-worker, csv, rules.\n' +
  '   Exports convertLocally(provider, file, password, config, onProgress) => Promise<Blob>.\n' +
  '   OCR_PROVIDERS = Set(["hsbccredit"]). For CSV: file.text(). For OCR: ocrPDFPages. For other PDF: extractPDFText.\n' +
  '   Export ConversionProgress type.\n\n' +
  '2. Update ' + ROOT + '/frontend/src/lib/components/UploadForm.svelte:\n' +
  '   Replace convertFile import with convertLocally from $lib/api.js.\n' +
  '   Add mergeConfigs import from $lib/rules.js.\n' +
  '   Add defaultConfig object.\n' +
  '   In handleSubmit: replace convertFile call with convertLocally.\n' +
  '   Keep all existing UI/CSS.',
  { description: 'Update frontend integration', subagent_type: 'general-purpose', model: 'sonnet' }
)

// Phase 1E: Vite config
phase('Vite Config')

const viteResult = await agent(
  'Update ' + ROOT + '/frontend/vite.config.js.\n' +
  'Add optimizeDeps: { include: ["pdfjs-dist", "tesseract.js"] }.\n' +
  'Remove the /convert proxy. Add build.target: "es2022".',
  { description: 'Update Vite config', subagent_type: 'general-purpose', model: 'haiku' }
)

return { summary: 'Phase 1 Foundation complete' }
