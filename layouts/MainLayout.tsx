import { Outlet } from "react-router-dom";
import CampusNav from "@/campus/components/CampusNav";

const MainLayout = () => {
  return (
    <section className="@container flex min-h-full w-full flex-col items-center">
      {/* 校园版顶部功能导航（好友/群/圈/话题区/推荐/主题分类/我的） */}
      <CampusNav />
      <div className="mx-auto w-full px-4 pb-8 pt-3 sm:px-6 md:pt-6">
        <Outlet />
      </div>
    </section>
  );
};

export default MainLayout;
