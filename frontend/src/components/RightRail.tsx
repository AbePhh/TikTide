import { featuredTopics } from "../constants/mock-data";

export function RightRail() {
  return (
    <aside className="right-rail">
      <section className="panel">
        <div className="panel-title-row">
          <h3>热榜话题</h3>
          <span>今日</span>
        </div>
        <div className="topic-list">
          {featuredTopics.map((topic, index) => (
            <article key={topic.id} className="topic-card">
              <div className="topic-rank">{index + 1}</div>
              <div>
                <div className="topic-title">#{topic.title}</div>
                <div className="topic-subtitle">{topic.subtitle}</div>
              </div>
              <div className="topic-heat">{topic.heat}</div>
            </article>
          ))}
        </div>
      </section>

      <section className="panel">
        <div className="panel-title-row">
          <h3>创作者数据</h3>
          <span>模拟</span>
        </div>
        <div className="metric-stack">
          <div className="metric-item">
            <span>今日播放</span>
            <strong>284,920</strong>
          </div>
          <div className="metric-item">
            <span>新增粉丝</span>
            <strong>1,248</strong>
          </div>
          <div className="metric-item">
            <span>互动率</span>
            <strong>8.7%</strong>
          </div>
        </div>
      </section>
    </aside>
  );
}
