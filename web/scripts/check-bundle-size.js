#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { promisify } = require('util');
const readdir = promisify(fs.readdir);
const stat = promisify(fs.stat);

// 번들 크기 제한 설정 (바이트)
const BUNDLE_LIMITS = {
  'index.js': 250 * 1024,      // 250KB
  'vendor.js': 500 * 1024,     // 500KB
  'index.css': 150 * 1024,     // 150KB
  'total': 1024 * 1024,        // 1MB
  'chunks': 100 * 1024,        // 100KB per chunk
};

// 경고 임계값 (제한의 90%)
const WARNING_THRESHOLD = 0.9;

// 디렉토리 크기 계산
async function getDirectorySize(dirPath) {
  let totalSize = 0;
  const files = await readdir(dirPath);
  
  for (const file of files) {
    const filePath = path.join(dirPath, file);
    const stats = await stat(filePath);
    
    if (stats.isDirectory()) {
      totalSize += await getDirectorySize(filePath);
    } else {
      totalSize += stats.size;
    }
  }
  
  return totalSize;
}

// 파일 크기 분석
async function analyzeBundle() {
  const distPath = path.join(__dirname, '..', 'dist');
  const assetsPath = path.join(distPath, 'assets');
  
  if (!fs.existsSync(distPath)) {
    console.error('❌ dist 디렉토리가 존재하지 않습니다. 먼저 빌드를 실행하세요.');
    process.exit(1);
  }
  
  const results = {
    total: 0,
    files: [],
    violations: [],
    warnings: [],
  };
  
  // assets 디렉토리의 모든 파일 분석
  if (fs.existsSync(assetsPath)) {
    const files = await readdir(assetsPath);
    
    for (const file of files) {
      const filePath = path.join(assetsPath, file);
      const stats = await stat(filePath);
      
      if (!stats.isDirectory()) {
        const size = stats.size;
        results.total += size;
        
        const fileInfo = {
          name: file,
          size: size,
          sizeKB: (size / 1024).toFixed(2),
          sizeMB: (size / 1024 / 1024).toFixed(2),
        };
        
        results.files.push(fileInfo);
        
        // 파일별 제한 체크
        if (file.includes('index') && file.endsWith('.js') && size > BUNDLE_LIMITS['index.js']) {
          results.violations.push({
            file: file,
            size: size,
            limit: BUNDLE_LIMITS['index.js'],
            type: 'error',
          });
        } else if (file.includes('vendor') && file.endsWith('.js') && size > BUNDLE_LIMITS['vendor.js']) {
          results.violations.push({
            file: file,
            size: size,
            limit: BUNDLE_LIMITS['vendor.js'],
            type: 'error',
          });
        } else if (file.endsWith('.css') && size > BUNDLE_LIMITS['index.css']) {
          results.violations.push({
            file: file,
            size: size,
            limit: BUNDLE_LIMITS['index.css'],
            type: 'error',
          });
        } else if (file.includes('chunk') && size > BUNDLE_LIMITS['chunks']) {
          results.violations.push({
            file: file,
            size: size,
            limit: BUNDLE_LIMITS['chunks'],
            type: 'error',
          });
        }
        
        // 경고 체크
        for (const [key, limit] of Object.entries(BUNDLE_LIMITS)) {
          if (key === 'total') continue;
          
          const warningLimit = limit * WARNING_THRESHOLD;
          if (
            (key === 'index.js' && file.includes('index') && file.endsWith('.js') && size > warningLimit && size <= limit) ||
            (key === 'vendor.js' && file.includes('vendor') && file.endsWith('.js') && size > warningLimit && size <= limit) ||
            (key === 'index.css' && file.endsWith('.css') && size > warningLimit && size <= limit) ||
            (key === 'chunks' && file.includes('chunk') && size > warningLimit && size <= limit)
          ) {
            results.warnings.push({
              file: file,
              size: size,
              limit: limit,
              threshold: warningLimit,
              type: 'warning',
            });
          }
        }
      }
    }
  }
  
  // 전체 크기 체크
  if (results.total > BUNDLE_LIMITS['total']) {
    results.violations.push({
      file: 'Total Bundle',
      size: results.total,
      limit: BUNDLE_LIMITS['total'],
      type: 'error',
    });
  } else if (results.total > BUNDLE_LIMITS['total'] * WARNING_THRESHOLD) {
    results.warnings.push({
      file: 'Total Bundle',
      size: results.total,
      limit: BUNDLE_LIMITS['total'],
      threshold: BUNDLE_LIMITS['total'] * WARNING_THRESHOLD,
      type: 'warning',
    });
  }
  
  return results;
}

// 결과 출력
function printResults(results) {
  console.log('\n📊 Bundle Size Analysis\n');
  console.log('═'.repeat(60));
  
  // 파일 목록
  console.log('\n📁 Files:');
  results.files
    .sort((a, b) => b.size - a.size)
    .forEach(file => {
      const emoji = file.name.endsWith('.js') ? '📜' : file.name.endsWith('.css') ? '🎨' : '📄';
      console.log(`  ${emoji} ${file.name.padEnd(40)} ${file.sizeKB.padStart(10)} KB`);
    });
  
  // 전체 크기
  console.log('\n📦 Total Size:');
  console.log(`  ${(results.total / 1024).toFixed(2)} KB (${(results.total / 1024 / 1024).toFixed(2)} MB)`);
  console.log(`  Limit: ${(BUNDLE_LIMITS['total'] / 1024 / 1024).toFixed(2)} MB`);
  
  // 경고
  if (results.warnings.length > 0) {
    console.log('\n⚠️  Warnings:');
    results.warnings.forEach(warning => {
      const percentage = ((warning.size / warning.limit) * 100).toFixed(1);
      console.log(`  ${warning.file}: ${(warning.size / 1024).toFixed(2)} KB (${percentage}% of limit)`);
    });
  }
  
  // 위반 사항
  if (results.violations.length > 0) {
    console.log('\n❌ Violations:');
    results.violations.forEach(violation => {
      const exceeded = violation.size - violation.limit;
      const percentage = ((violation.size / violation.limit) * 100).toFixed(1);
      console.log(`  ${violation.file}:`);
      console.log(`    Size: ${(violation.size / 1024).toFixed(2)} KB`);
      console.log(`    Limit: ${(violation.limit / 1024).toFixed(2)} KB`);
      console.log(`    Exceeded by: ${(exceeded / 1024).toFixed(2)} KB (${percentage}% of limit)`);
    });
  }
  
  console.log('\n' + '═'.repeat(60));
  
  // 상태 요약
  if (results.violations.length === 0) {
    if (results.warnings.length === 0) {
      console.log('\n✅ All bundle sizes are within limits!');
    } else {
      console.log('\n⚠️  Bundle sizes are approaching limits.');
    }
  } else {
    console.log('\n❌ Bundle size limits exceeded!');
  }
}

// 메인 실행
async function main() {
  try {
    const results = await analyzeBundle();
    printResults(results);
    
    // CI에서 실패 처리
    if (results.violations.length > 0) {
      process.exit(1);
    }
  } catch (error) {
    console.error('❌ Error analyzing bundle:', error);
    process.exit(1);
  }
}

// 스크립트 실행
if (require.main === module) {
  main();
}