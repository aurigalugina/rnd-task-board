<script lang="ts">
  import { TrendingUp, AlertCircle, CheckCircle2, Activity } from 'lucide-svelte';
  
  export let label: string;
  export let value: string | number;
  export let tone: 'warn' | 'good' | 'accent' | undefined = undefined;
  
  // Pick icon based on tone
  const iconMap: Record<string, any> = {
    'good': CheckCircle2,
    'warn': AlertCircle,
    'accent': TrendingUp,
  };
  
  const IconComponent = tone && iconMap[tone] ? iconMap[tone] : Activity;
</script>

<div class="stat-card-container {tone ? `tone-${tone}` : ''}">
  <div class="stat-card-header">
    <div class="stat-icon">
      <svelte:component this={IconComponent} size={16} />
    </div>
    <div class="stat-label">{label}</div>
  </div>
  <div class="stat-value mono">{value}</div>
</div>

<style>
  .stat-card-container {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 14px 12px;
    border-radius: 12px;
    background: var(--content-bg);
    border: 1px solid rgba(0, 0, 0, 0.05);
    cursor: pointer;
    position: relative;
    overflow: hidden;
  }
  
  .stat-card-container::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 3px;
    background: var(--win-blue);
    opacity: 0;
    transition: opacity 0.3s ease;
  }
  
  .stat-card-container:hover::before {
    opacity: 1;
  }
  
  .stat-card-header {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  
  .stat-icon {
    width: 20px;
    height: 20px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--win-blue);
    flex-shrink: 0;
  }
  
  .stat-label {
    font-size: 10.5px;
    color: var(--text-muted);
    font-weight: 500;
    letter-spacing: 0.3px;
    line-height: 1.2;
  }
  
  .stat-value {
    font-size: 16px;
    font-weight: 700;
    color: var(--text-primary);
  }
  
  /* Tone variations */
  .stat-card-container.tone-good {
    border-color: rgba(14, 165, 116, 0.2);
    background: rgba(14, 165, 116, 0.05);
  }
  
  .stat-card-container.tone-good .stat-icon {
    color: var(--win-green);
  }
  
  .stat-card-container.tone-good::before {
    background: var(--win-green);
  }
  
  .stat-card-container.tone-warn {
    border-color: rgba(239, 68, 68, 0.2);
    background: rgba(239, 68, 68, 0.05);
  }
  
  .stat-card-container.tone-warn .stat-icon {
    color: var(--win-red);
  }
  
  .stat-card-container.tone-warn::before {
    background: var(--win-red);
  }
  
  .stat-card-container.tone-warn .stat-value {
    color: var(--win-red);
  }
  
  .stat-card-container.tone-accent {
    border-color: rgba(124, 58, 237, 0.2);
    background: rgba(124, 58, 237, 0.05);
  }
  
  .stat-card-container.tone-accent .stat-icon {
    color: var(--win-blue);
  }
  
  .stat-card-container.tone-accent::before {
    background: var(--win-blue);
  }
  
  .stat-card-container.tone-accent .stat-value {
    color: var(--win-blue);
  }
</style>
